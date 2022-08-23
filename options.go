package ihttp

import "errors"

// Options represent the flags.
type Options struct {
	JSON    bool
	Form    bool
	Verbose bool
	scheme  string
}

// Scheme return the value of the scheme unexported field by defalut will
// return `http` if the scheme is an empty string.
func (o *Options) Scheme() string {
	if o.scheme == "" {
		return "http"
	}
	return o.scheme
}

// SetScheme set value for the scheme unexported field. Its value can be
// obtained with the Scheme method.
func (o *Options) SetScheme(s string) {
	o.scheme = s
}

// IsValid validate field values.
func (o *Options) IsValid() error {
	if o.JSON && o.Form {
		return errors.New("you canot specify both of -json and -form")
	}
	return nil
}
