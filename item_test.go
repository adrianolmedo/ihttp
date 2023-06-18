package ihttp

import (
	"reflect"
	"testing"
)

// argBench is an example of HTTP Header.
var argBench = "Authorization:eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJsb2dnZWRJbkFzIjoiYWRtaW4iLCJpYXQiOjE0MjI3Nzk2Mzh9.gzSraSYS8EXBxLN_oWnFSRgCzcmJmMjLiuyu5CSpyHI"

// sepsTest slice with all separators.
var sepsTest = SepsGroupAllItems()

// TestTokenize with SepsGroupAllItems() by default.
func TestTokenize(t *testing.T) {
	tt := []struct {
		name string
		arg  string
		want []interface{}
	}{
		{
			name: "escape separator 1",
			arg:  `foo\=bar\\baz`,
			want: []interface{}{"foo", []byte("="), "bar\\\\baz"}},
		{
			name: "escape separator 2",
			arg:  `foo\:bar:baz`,
			want: []interface{}{"foo", []byte(":"), "bar:baz"}},
		{
			// Backslash before non special character does not escaped
			name: "escape separator 3",
			arg:  "path\\==c:\\windows",
			want: []interface{}{"path", []byte("="), "=c:\\windows"},
		},
		{
			// Backslash before non special character does not escaped
			name: "does not escape 1",
			arg:  "path=c:\\windows",
			want: []interface{}{"path=c:\\windows"},
		},
		{
			// Backslash before non special character does not escaped
			name: "does not escape 2",
			arg:  "path=c:\\windows\\",
			want: []interface{}{"path=c:\\windows\\"},
		},
		{
			name: "escape longsep",
			arg:  `bob\:==foo`,
			want: []interface{}{"bob", []byte(":"), "==foo"},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			got := tokenize(tc.arg, sepsTest)
			if !reflect.DeepEqual(tc.want, got) {
				t.Errorf("%s %s\nwant\t%v\ngot\t%v", tc.name, tc.arg, tc.want, got)
			}
		})
	}
}

func BenchmarkTokenize(b *testing.B) {
	for n := 0; n < b.N; n++ {
		tokenize(argBench, sepsTest)
	}
}

func TestParseItem(t *testing.T) {
	tt := []struct {
		name        string
		arg         string
		want        item
		errExpected bool
	}{
		{
			name:        "escape separator 1",
			arg:         `foo\=bar\\baz`,
			want:        item{},
			errExpected: true,
		},
		{
			name: "escape separator 2",
			arg:  `foo\=bar:baz`,
			want: item{
				Key: "foo=bar",
				Val: "baz",
				Sep: ":",
				Arg: "foo\\=bar:baz",
			},
			errExpected: false,
		},
		{
			name: "escape separator 3",
			arg:  "path\\==c:\\windows",
			want: item{
				Key: "path=",
				Val: "c:\\windows",
				Sep: "=",
				Arg: "path\\==c:\\windows",
			},
			errExpected: false,
		},
		{
			// Backslash before non special character does not escaped.
			name: "does not escape 1",
			arg:  "path=c:\\windows",
			want: item{
				Key: "path",
				Val: "c:\\windows",
				Sep: "=",
				Arg: "path=c:\\windows",
			},
			errExpected: false,
		},
		{
			// Backslash before non special character does not escaped.
			name: "does not escape 2",
			arg:  "path=c:\\windows\\",
			want: item{
				Key: "path",
				Val: "c:\\windows\\",
				Sep: "=",
				Arg: "path=c:\\windows\\",
			},
			errExpected: false,
		},
		{
			name: "escape longsep",
			arg:  `bob\:==foo`,
			want: item{
				Key: "bob:",
				Val: "foo",
				Sep: "==",
				Arg: "bob\\:==foo",
			},
			errExpected: false,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseItem(tc.arg, sepsTest)
			errReceived := err != nil

			if errReceived != tc.errExpected {
				t.Fatalf("%s: %s: unexpected error status: %v", tc.name, tc.arg, err)
			}

			if !tc.errExpected && !reflect.DeepEqual(tc.want, got) {
				t.Errorf("%s\nwant\t%#v\ngot\t%#v", tc.arg, tc.want, got)
			}
		})
	}
}

func BenchmarkParseItem(b *testing.B) {
	for n := 0; n < b.N; n++ {
		parseItem(argBench, sepsTest)
	}
}
