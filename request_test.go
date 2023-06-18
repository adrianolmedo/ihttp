package ihttp

import (
	"net/http"
	"net/url"
	"reflect"
	"testing"
)

func TestParseRequestBody(t *testing.T) {
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

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			inp, err := NewInput(tc.args, nil)
			if err != nil {
				t.Fatal(err)
			}

			got, err := parseRequestBody(inp)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(tc.want, got) {
				t.Errorf("%s\nwant\t%v\ngot\t%v", tc.name, tc.want, got)
			}
		})
	}
}

func TestParseHeaders(t *testing.T) {
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

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			inp, err := NewInput(tc.args, nil)
			if err != nil {
				t.Fatal(err)
			}

			req, err := http.NewRequest(inp.Method, inp.URL, nil)
			if err != nil {
				t.Fatal(err)
			}

			r := &request{req}

			err = r.parseHeaders(inp)
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

// TestParseParams
func TestParseQuery(t *testing.T) {
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

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			inp, err := NewInput(tc.args, nil)
			if err != nil {
				t.Fatal(err)
			}

			req, err := http.NewRequest(inp.Method, inp.URL, nil)
			if err != nil {
				t.Fatal(err)
			}

			r := &request{req}

			if err := r.parseQuery(inp); err != nil {
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
