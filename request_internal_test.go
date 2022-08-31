package ihttp

import (
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
				contentType: "application/json",
			},
		},
		{
			name: "data=field query==value",
			args: []string{"url", "data=field", "query==value"},
			want: bodyTuple{
				content:     nil,
				contentType: "application/json",
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

/*func TestParseRequestHeaders(t *testing.T) {
	tt := []struct {
		name        string
		args        []string
		want        http.Header
		errExpected bool
	}{
		{
			name: "Header:value",
			args: []string{"url", "Header:value"},
			want: http.Header{
				"Header": []string{"value"},
			},
			errExpected: false,
		},
		{
			name: "Unset-Header:",
			args: []string{"url", "Unset-Header:"},
			want: http.Header{
				"Unset-Header": []string{""},
			},
			errExpected: false,
		},
		{
			name:        "Empty-Header;",
			args:        []string{"url", "Empty-Header;"},
			want:        http.Header{},
			errExpected: true,
		},
		{
			name: "escape header separator",
			args: []string{"url", `foo\:bar:baz`},
			want: http.Header{
				"foo:bar": []string{"baz"},
			},
			errExpected: false,
		},
		{
			name: "escape file upload separator",
			args: []string{"url", `jack\@jill:hill`},
			want: http.Header{
				"jack@jill": []string{"hill"},
			},
			errExpected: false,
		},
		{
			name: "query==value Header:value",
			args: []string{"url", "query==value", "Header:value"},
			want: http.Header{
				"Header": []string{"value"},
			},
			errExpected: false,
		},
	}

	p := New()
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			err := p.parseItems(tc.args)
			if err != nil {
				t.Fatal(err)
			}

			p.req, err = http.NewRequest("GET", "httpbingo.org/get", nil)
			if err != nil {
				t.Fatal(err)
			}

			err = p.parseRequestHeaders()
			if (err != nil) != tc.errExpected {
				t.Fatalf("%s: unexpected error status: %v", tc.name, err)
			}

			got := p.req.Header
			if !tc.errExpected && !reflect.DeepEqual(tc.want, got) {
				t.Errorf("%s\nwant\t%#v\ngot\t%#v", tc.name, tc.want, got)
			}
		})
	}
}*/

// TestParseParams
/*func TestParseRequestQuery(t *testing.T) {
	tt := []struct {
		name string
		args []string
		want url.Values
	}{
		{
			name: "query==value",
			args: []string{"url", "query==value"},
			want: url.Values{
				"query": []string{"value"},
			},
		},
		{
			name: "data=field query==value",
			args: []string{"url", "data=field", "query==value"},
			want: url.Values{
				"query": []string{"value"},
			},
		},
	}

	p := New()
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			err := p.parseItems(tc.args)
			if err != nil {
				t.Fatal(err)
			}

			p.req, err = http.NewRequest("GET", "httpbingo.org/get", nil)
			if err != nil {
				t.Fatal(err)
			}

			p.parseRequestQuery()
			//fmt.Printf("URL      %+v\n", p.Request.URL)          // httpbingo.org/get?query=value
			//fmt.Printf("RawQuery %+v\n", p.Request.URL.RawQuery) // query=value
			//fmt.Printf("Query    %+v\n", p.Request.URL.Query())  // map[query:[value]]
			got := p.req.URL.Query()
			if !reflect.DeepEqual(tc.want, got) {
				t.Errorf("%s want %#v got %#v", tc.name, tc.want, got)
			}
		})
	}
}*/
