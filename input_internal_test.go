package ihttp

import (
	"reflect"
	"testing"
)

func TestProcessURL(t *testing.T) {
	tt := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "http localhost",
			args: []string{"localhost"},
			want: "http://localhost",
		},
		{
			name: "http leading colon slash slash",
			args: []string{"://domain.xxx/get"},
			want: "http://domain.xxx/get",
		},
		{
			name: "localhost shorthand",
			args: []string{":"},
			want: "http://localhost",
		},
		{
			name: "localhost shorthand with slash",
			args: []string{":/"},
			want: "http://localhost/",
		},
		{
			name: "localhost shorthand with port",
			args: []string{":3000"},
			want: "http://localhost:3000",
		},
		{
			name: "localhost shorthand with path",
			args: []string{":/path"},
			want: "http://localhost/path",
		},
		{
			name: "localhost shorthand with port and slash",
			args: []string{":3000/"},
			want: "http://localhost:3000/",
		},
		{
			name: "localhost shorthand with port and path",
			args: []string{":3000/path"},
			want: "http://localhost:3000/path",
		},
		{
			name: "ipv6 as shorthand",
			args: []string{"::1"},
			want: "http://::1",
		},
		{
			name: "longer ipv6 as shorthand",
			args: []string{"::ffff:c000:0280"},
			want: "http://::ffff:c000:0280",
		},
		{
			name: "full ipv6 as shorthand",
			args: []string{"0000:0000:0000:0000:0000:0000:0000:0001"},
			want: "http://0000:0000:0000:0000:0000:0000:0000:0001",
		},
	}

	inp := &Input{}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			err := inp.processURL(tc.args[0])
			if err != nil {
				t.Fatal(err)
			}

			got := inp.URL
			if tc.want != got {
				t.Errorf("\narg: %s\n%s: want %q, got %q,", tc.args[0], tc.name, tc.want, got)
			}
		})
	}
}

func TestProcessURLWithHTTPS(t *testing.T) {
	tt := []struct {
		name   string
		args   []string
		scheme string
		want   string
	}{
		{
			name:   "https localhost",
			args:   []string{"localhost"},
			scheme: "https",
			want:   "https://localhost",
		},
		{
			name:   "https leading colon slash slash",
			args:   []string{"://domain.xxx/get"},
			scheme: "https",
			want:   "https://domain.xxx/get",
		},
	}

	opts := Options{}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			opts.SetScheme(tc.scheme)
			inp := Input{Options: opts}

			err := inp.processURL(tc.args[0])
			if err != nil {
				t.Fatal(err)
			}

			got := inp.URL
			if tc.want != got {
				t.Errorf("\narg: %s\n%s: want %q, got %q,", tc.args[0], tc.name, tc.want, got)
			}
		})
	}
}

func TestProcessItems(t *testing.T) {
	tt := []struct {
		name        string
		args        []string
		want        []item
		errExpected bool
	}{
		{
			name: "data=field test:header",
			args: []string{"data=field", "test:header"},
			want: []item{
				{
					Key: "data",
					Val: "field",
					Sep: "=",
					Arg: "data=field",
				},
				{
					Key: "test",
					Val: "header",
					Sep: ":",
					Arg: "test:header",
				},
			},
		},
		{
			name:        "Error localhost",
			args:        []string{"query==value", "localhost"},
			errExpected: true,
		},
		{
			name: "query==value",
			args: []string{"localhost", "query==value"},
			want: []item{
				{
					Key: "query",
					Val: "value",
					Sep: "==",
					Arg: "query==value",
				},
			},
			errExpected: true,
		},
	}

	inp := &Input{}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			err := inp.processItems(tc.args)
			if (err != nil) != tc.errExpected {
				t.Fatalf("%s: unexpected error status: %v", tc.name, err)
			}

			got := inp.Items
			if !tc.errExpected && !reflect.DeepEqual(tc.want, got) {
				t.Errorf("%s\nwant\t%#v\ngot\t%#v", tc.args, tc.want, got)
			}
		})
	}
}
