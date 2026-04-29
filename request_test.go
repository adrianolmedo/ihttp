package ihttp

import (
	"encoding/json"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

func TestParseKey(t *testing.T) {
	tt := []struct {
		name        string
		input       string
		want        []keyPath
		errExpected bool
	}{
		{
			name:  "simple key",
			input: "foo",
			want: []keyPath{
				{value: "foo", escaped: false},
			},
		},
		{
			name:  "nested brackets",
			input: "foo[bar][baz]",
			want: []keyPath{
				{value: "foo", escaped: false},
				{value: "bar", escaped: false},
				{value: "baz", escaped: false},
			},
		},
		{
			name:  "numeric index",
			input: "foo[0]",
			want: []keyPath{
				{value: "foo", escaped: false},
				{value: "0", escaped: false},
			},
		},
		{
			name:  "escaped numeric treated as string",
			input: `foo[\1]`,
			want: []keyPath{
				{value: "foo", escaped: false},
				{value: "1", escaped: true},
			},
		},
		{
			name:  "escaped bracket literal",
			input: `foo\[bar\]`,
			want: []keyPath{
				{value: "foo[bar]", escaped: true},
			},
		},
		{
			name:        "missing closing bracket",
			input:       "foo[bar",
			want:        nil,
			errExpected: true,
		},
		{
			name:        "unexpected closing bracket",
			input:       "foo]bar",
			want:        nil,
			errExpected: true,
		},
		{
			name:        "escape at end of string",
			input:       `foo\`,
			want:        nil,
			errExpected: true,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseKey(tc.input)
			if (err != nil) != tc.errExpected {
				t.Fatalf("unexpected error status: %v", err)
			}
			if !tc.errExpected && !reflect.DeepEqual(tc.want, got) {
				t.Errorf("\ngot\t%#v\nwant\t%#v", got, tc.want)
			}
		})
	}
}

func TestInsertJSON(t *testing.T) {
	tt := []struct {
		name  string
		start any
		path  []keyPath
		value any
		want  any
	}{
		{
			name:  "flat key",
			path:  []keyPath{{value: "foo"}},
			value: "bar",
			want:  map[string]any{"foo": "bar"},
		},
		{
			name:  "numeric index creates array",
			path:  []keyPath{{value: "0"}},
			value: "x",
			want:  []any{"x"},
		},
		{
			name:  "escaped numeric creates map key",
			path:  []keyPath{{value: "1", escaped: true}},
			value: "stringified",
			want:  map[string]any{"1": "stringified"},
		},
		{
			name: "nested map",
			path: []keyPath{
				{value: "a"},
				{value: "b"},
			},
			value: 42,
			want:  map[string]any{"a": map[string]any{"b": 42}},
		},
		{
			name: "append to array with empty segment",
			path: []keyPath{
				{value: "arr"},
				{value: ""},
			},
			value: "item",
			want:  map[string]any{"arr": []any{"item"}},
		},
		{
			name: "sparse array pads with nil",
			path: []keyPath{
				{value: "arr"},
				{value: "2"},
			},
			value: "z",
			want:  map[string]any{"arr": []any{nil, nil, "z"}},
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			// insertJSON expects the root to be wrapped one level up,
			// so we drive it the same way buildJSONBody does: start from nil
			// and insert at the top-level path.
			got, err := insertJSON(tc.start, tc.path, tc.value)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(tc.want, got) {
				t.Errorf("\ngot\t%#v\nwant\t%#v", got, tc.want)
			}
		})
	}
}

func TestBuildJSONBody(t *testing.T) {
	tt := []struct {
		name string
		args []string
		want any
	}{
		{
			name: "append to array",
			args: []string{"", "bottle-on-wall[]:=1", "bottle-on-wall[]:=2", "bottle-on-wall[]:=3"},
			want: map[string]any{
				"bottle-on-wall": []any{float64(1), float64(2), float64(3)},
			},
		},
		{
			name: "mixed nested map and indexed array",
			args: []string{
				"",
				"pet[species]=Dahut",
				`pet[name]:="Hypatia"`,
				"kids[1]=Thelma",
				`kids[0]:="Ashley"`,
			},
			want: map[string]any{
				"pet":  map[string]any{"species": "Dahut", "name": "Hypatia"},
				"kids": []any{"Ashley", "Thelma"},
			},
		},
		{
			name: "array of objects",
			args: []string{
				"",
				"pet[0][species]=Dahut",
				"pet[0][name]=Hypatia",
				"pet[1][species]=Felis Stultus",
				`pet[1][name]:="Billie"`,
			},
			want: map[string]any{
				"pet": []any{
					map[string]any{"species": "Dahut", "name": "Hypatia"},
					map[string]any{"species": "Felis Stultus", "name": "Billie"},
				},
			},
		},
		{
			name: "deeply nested with sparse array",
			args: []string{"", "wow[such][deep][3][much][power][!]=Amaze"},
			want: map[string]any{
				"wow": map[string]any{
					"such": map[string]any{
						"deep": []any{
							nil, nil, nil,
							map[string]any{
								"much": map[string]any{
									"power": map[string]any{"!": "Amaze"},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "mixed append and index",
			args: []string{"", "mix[]=scalar", "mix[2]=something", `mix[4]:="something 2"`},
			want: map[string]any{
				"mix": []any{"scalar", nil, "something", nil, "something 2"},
			},
		},
		{
			name: "single append",
			args: []string{"", "highlander[]=one"},
			want: map[string]any{
				"highlander": []any{"one"},
			},
		},
		{
			name: "escaped bracket in key literal",
			args: []string{"", "error[good]=BOOM!", `error\[bad:="BOOM BOOM!"`},
			want: map[string]any{
				"error":     map[string]any{"good": "BOOM!"},
				"error[bad": "BOOM BOOM!",
			},
		},
		{
			name: "special JSON values in array",
			args: []string{
				"",
				"special[]:=true",
				"special[]:=false",
				`special[]:="true"`,
				"special[]:=null",
			},
			want: map[string]any{
				"special": []any{true, false, "true", nil},
			},
		},
		{
			name: "fully escaped bracket keys",
			args: []string{
				"",
				`\[\]:=1`,
				`escape\[d\]:=1`,
				`escaped\[\]:=1`,
				`e\[s\][c][a][p][\[ed\]][]:=1`,
			},
			want: map[string]any{
				"[]":        float64(1),
				"escape[d]": float64(1),
				"escaped[]": float64(1),
				"e[s]": map[string]any{
					"c": map[string]any{
						"a": map[string]any{
							"p": map[string]any{
								"[ed]": []any{float64(1)},
							},
						},
					},
				},
			},
		},
		{
			name: "root array",
			args: []string{"", "[]:=1", "[]=foo"},
			want: []any{float64(1), "foo"},
		},
		{
			name: "escaped nonbracket characters in keys",
			args: []string{"", `\]:=1`, `\[\]1:=1`, `\[1\]\]:=1`},
			want: map[string]any{
				"]":    float64(1),
				"[]1":  float64(1),
				"[1]]": float64(1),
			},
		},
		{
			name: "escaped and unescaped brackets in same key",
			args: []string{
				"",
				`foo\[bar\][baz]:=1`,
				`foo\[bar\]\[baz\]:=3`,
				`foo[bar][\[baz\]]:=4`,
			},
			want: map[string]any{
				"foo[bar]":      map[string]any{"baz": float64(1)},
				"foo[bar][baz]": float64(3),
				"foo": map[string]any{
					"bar": map[string]any{
						"[baz]": float64(4),
					},
				},
			},
		},
		{
			name: "nested appends",
			args: []string{"", "key[]:=1", "key[][]:=2", "key[][][]:=3", "key[][][]:=4"},
			want: map[string]any{
				"key": []any{float64(1), []any{float64(2)}, []any{[]any{float64(3)}}, []any{[]any{float64(4)}}},
			},
		},
		{
			name: "index then appends",
			args: []string{"", "x[0]:=1", "x[]:=2", "x[]:=3", "x[][]:=4", "x[][]:=5"},
			want: map[string]any{
				"x": []any{float64(1), float64(2), float64(3), []any{float64(4)}, []any{float64(5)}},
			},
		},
		{
			name: "complex bar baz with index and append mixing",
			args: []string{
				":",
				"foo[bar][5][]:=5",
				"foo[bar][]:=6",
				"foo[bar][][]:=7",
				"foo[bar][][x]=dfasfdas",
				`foo[baz]:=[1, 2, 3]`,
				"foo[baz][]:=4",
			},
			want: map[string]any{
				"foo": map[string]any{
					"bar": []any{
						nil, nil, nil, nil, nil,
						[]any{float64(5)},
						float64(6),
						[]any{float64(7)},
						map[string]any{"x": "dfasfdas"},
					},
					"baz": []any{float64(1), float64(2), float64(3), float64(4)},
				},
			},
		},
		{
			name: "append then indexed merge and nested escape",
			args: []string{
				":",
				"foo[]:=1",
				"foo[]:=2",
				"foo[][key]=value",
				"foo[2][key 2]=value 2",
				`foo[2][key \[]=value 3`,
				`bar[nesting][under][!][empty][?][\\key]:=4`,
			},
			want: map[string]any{
				"foo": []any{
					float64(1),
					float64(2),
					map[string]any{
						"key":   "value",
						"key 2": "value 2",
						"key [": "value 3",
					},
				},
				"bar": map[string]any{
					"nesting": map[string]any{
						"under": map[string]any{
							"!": map[string]any{
								"empty": map[string]any{
									"?": map[string]any{
										`\key`: float64(4),
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "escaped brackets as literal keys and nested escape combos",
			args: []string{
				":",
				`foo\[key\]:=1`,
				`bar\[1\]:=2`,
				`quux[key\[escape\]]:=4`,
				`quux[key 2][\\][\\\\][\\\[\]\\\]\\\[\n\\]:=5`,
			},
			want: map[string]any{
				"foo[key]": float64(1),
				"bar[1]":   float64(2),
				"quux": map[string]any{
					"key[escape]": float64(4),
					"key 2": map[string]any{
						`\`: map[string]any{
							`\\`: map[string]any{
								`\[]\]\[` + "n" + `\`: float64(5),
							},
						},
					},
				},
			},
		},
	}

	opts := Options{}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			in, err := NewInput(tc.args, opts)
			if err != nil {
				t.Fatal(err)
			}
			got, err := buildJSONBody(in.Items)
			if err != nil {
				t.Fatal(err)
			}
			// Unmarshal the JSON bytes back into any for comparison.
			var gotAny any
			if err := json.Unmarshal(got.content, &gotAny); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(tc.want, gotAny) {
				t.Errorf("\ngot\t%#v\nwant\t%#v", gotAny, tc.want)
			}
		})
	}
}

func TestBuildRequestBody(t *testing.T) {
	tt := []struct {
		name        string
		args        []string
		opts        Options
		wantType    string
		errExpected bool
	}{
		{
			name:     "url only",
			args:     []string{":"},
			opts:     Options{},
			wantType: "",
		},
		{
			name:     "url foo=bar",
			args:     []string{":", "foo=bar"},
			opts:     Options{},
			wantType: "application/json",
		},
		{
			name:     "-json url",
			args:     []string{":"},
			opts:     Options{JSON: true},
			wantType: "application/json",
		},
		{
			name:     "-form url foo=bar",
			args:     []string{":", "foo=bar"},
			opts:     Options{Form: true},
			wantType: "application/x-www-form-urlencoded; charset=utf-8",
		},
		{
			name:     "-multipart url foo=bar",
			args:     []string{":", "foo=bar"},
			opts:     Options{Multipart: true},
			wantType: "multipart/form-data",
		},
		{
			name:     "-form url file upload",
			args:     []string{":", "file@examples/plain.txt"},
			opts:     Options{Form: true},
			wantType: "multipart/form-data",
		},
		{
			name:     "-raw url",
			args:     []string{":"},
			opts:     Options{Raw: `{"foo":"bar"}`},
			wantType: "application/json",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			inp, err := NewInput(tc.args, tc.opts)
			if err != nil {
				t.Fatal(err)
			}
			got, err := buildBody(inp)
			if (err != nil) != tc.errExpected {
				t.Fatalf("unexpected error status: %v", err)
			}
			if tc.errExpected {
				return
			}
			// For multipart the boundary is random, so only check the prefix.
			if tc.opts.Multipart || (tc.opts.Form && containsFile(inp.Items)) {
				if !strings.HasPrefix(got.contentType, "multipart/form-data") {
					t.Errorf("want content type prefixed %q, got %q", tc.wantType, got.contentType)
				}
				return
			}
			if got.contentType != tc.wantType {
				t.Errorf("want content type %q, got %q", tc.wantType, got.contentType)
			}
		})
	}
}

func containsFile(items []item) bool {
	for _, it := range items {
		if it.Sep == SepFileUpload {
			return true
		}
	}
	return false
}

func TestBuildHeaders(t *testing.T) {
	tt := []struct {
		name        string
		args        []string
		want        http.Header
		errExpected bool
	}{
		{
			name: "Header:value",
			args: []string{"httpbingo.org/get", "Header:value"},
			want: http.Header{
				"Header": []string{"value"},
			},
			errExpected: false,
		},
		{
			name: "Unset-Header:",
			args: []string{"httpbingo.org/get", "Unset-Header:"},
			want: http.Header{
				"Unset-Header": []string{""},
			},
			errExpected: false,
		},
		{
			name:        "Empty-Header;",
			args:        []string{"httpbingo.org/get", "Empty-Header;"},
			want:        http.Header{},
			errExpected: true,
		},
		{
			name: "escape header separator",
			args: []string{"httpbingo.org/get", `foo\:bar:baz`},
			want: http.Header{
				"foo:bar": []string{"baz"},
			},
			errExpected: false,
		},
		{
			name: "escape file upload separator",
			args: []string{"httpbingo.org/get", `jack\@jill:hill`},
			want: http.Header{
				"jack@jill": []string{"hill"},
			},
			errExpected: false,
		},
		{
			name: "query==value Header:value",
			args: []string{"httpbingo.org/get", "query==value", "Header:value"},
			want: http.Header{
				"Header": []string{"value"},
			},
			errExpected: false,
		},
	}
	opts := Options{}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			inp, err := NewInput(tc.args, opts)
			if err != nil {
				t.Fatal(err)
			}
			req, err := http.NewRequest(inp.Method, inp.URL, nil)
			if err != nil {
				t.Fatal(err)
			}
			r := &request{req}
			err = r.buildHeaders(inp)
			if (err != nil) != tc.errExpected {
				t.Fatalf("%s: unexpected error status: %v", tc.name, err)
			}
			got := r.Request.Header
			if !tc.errExpected && !reflect.DeepEqual(tc.want, got) {
				t.Errorf("%s\ngot\t%#v\nwant\t%#v", tc.name, got, tc.want)
			}
		})
	}
}

func TestBuildURLQuery(t *testing.T) {
	tt := []struct {
		name string
		args []string
		want url.Values
	}{
		{
			name: "query==value",
			args: []string{"httpbingo.org/get", "query==value"},
			want: url.Values{
				"query": []string{"value"},
			},
		},
		{
			name: "data=field query==value",
			args: []string{"httpbingo.org/get", "data=field", "query==value"},
			want: url.Values{
				"query": []string{"value"},
			},
		},
	}
	opts := Options{}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			inp, err := NewInput(tc.args, opts)
			if err != nil {
				t.Fatal(err)
			}
			req, err := http.NewRequest(inp.Method, inp.URL, nil)
			if err != nil {
				t.Fatal(err)
			}
			r := &request{req}
			if err := r.buildURLQuery(inp); err != nil {
				t.Fatal(err)
			}
			//fmt.Printf("URL      %+v\n", r.Request.URL)          // httpbingo.org/get?query=value
			//fmt.Printf("RawQuery %+v\n", r.Request.URL.RawQuery) // query=value
			//fmt.Printf("Query    %+v\n", r.Request.URL.Query())  // map[query:[value]]
			got := r.Request.URL.Query()
			if !reflect.DeepEqual(tc.want, got) {
				t.Errorf("%s got %#v want %#v", tc.name, got, tc.want)
			}
		})
	}
}
