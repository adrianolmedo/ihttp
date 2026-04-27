package ihttp

import (
	"net/http"
	"net/url"
	"reflect"
	"testing"
)

func TestParseKey(t *testing.T) {
	tt := []struct {
		name        string
		input       string
		want        []pathSegment
		errExpected bool
	}{
		{
			name:  "simple key",
			input: "foo",
			want: []pathSegment{
				{value: "foo", escaped: false},
			},
			errExpected: false,
		},
		{
			name:  "nested brackets",
			input: "foo[bar][baz]",
			want: []pathSegment{
				{value: "foo", escaped: false},
				{value: "bar", escaped: false},
				{value: "baz", escaped: false},
			},
			errExpected: false,
		},
		{
			name:  "numeric index",
			input: "foo[0]",
			want: []pathSegment{
				{value: "foo", escaped: false},
				{value: "0", escaped: false},
			},
			errExpected: false,
		},
		{
			name:  "escaped numeric treated as string",
			input: `foo[\1]`,
			want: []pathSegment{
				{value: "foo", escaped: false},
				{value: "1", escaped: true},
			},
			errExpected: false,
		},
		{
			name:  "escaped bracket literal",
			input: `foo\[bar\]`,
			want: []pathSegment{
				{value: "foo[bar]", escaped: true},
			},
			errExpected: false,
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
				t.Errorf("\nwant\t%#v\ngot\t%#v", tc.want, got)
			}
		})
	}
}

func TestInsertJSON(t *testing.T) {
	tt := []struct {
		name  string
		start any
		path  []pathSegment
		value any
		want  any
	}{
		{
			name:  "flat key",
			start: nil,
			path:  []pathSegment{{value: "foo"}},
			value: "bar",
			want:  map[string]any{"foo": "bar"},
		},
		{
			name:  "numeric index creates array",
			start: nil,
			path:  []pathSegment{{value: "0"}},
			value: "x",
			want:  []any{"x"},
		},
		{
			name:  "escaped numeric creates map key",
			start: nil,
			path:  []pathSegment{{value: "1", escaped: true}},
			value: "stringified",
			want:  map[string]any{"1": "stringified"},
		},
		{
			name:  "nested map",
			start: nil,
			path: []pathSegment{
				{value: "a"},
				{value: "b"},
			},
			value: 42,
			want:  map[string]any{"a": map[string]any{"b": 42}},
		},
		{
			name:  "append to array with empty segment",
			start: nil,
			path: []pathSegment{
				{value: "arr"},
				{value: ""},
			},
			value: "item",
			want:  map[string]any{"arr": []any{"item"}},
		},
		{
			name:  "sparse array pads with nil",
			start: nil,
			path: []pathSegment{
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
				t.Errorf("\nwant\t%#v\ngot\t%#v", tc.want, got)
			}
		})
	}
}

func TestBuildRequestBody(t *testing.T) {
	tt := []struct {
		name string
		args []string
		want bodyTuple
	}{
		{
			name: "data=field",
			args: []string{"url", "data=field"},
			want: bodyTuple{
				content:     nil,
				contentType: "",
			},
		},
		{
			name: "data=field query==value",
			args: []string{"url", "data=field", "query==value"},
			want: bodyTuple{
				content:     nil,
				contentType: "",
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
			got, err := buildBody(inp)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(tc.want, got) {
				t.Errorf("%s\nwant\t%v\ngot\t%v", tc.name, tc.want, got)
			}
		})
	}
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
				t.Errorf("%s\nwant\t%#v\ngot\t%#v", tc.name, tc.want, got)
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
				t.Errorf("%s want %#v got %#v", tc.name, tc.want, got)
			}
		})
	}
}
