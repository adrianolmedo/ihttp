package main

import (
	"encoding/json"
	"fmt"

	"github.com/adrianolmedo/ihttp"
)

// dbug debug output when Options.Debug is true.
type dbug struct {
	opts ihttp.Options
	inp  *ihttp.Input
}

// opts only debug output of ihttp.Options.
type opts struct {
	Scheme    string
	JSON      bool
	Form      bool
	Multipart bool
	Raw       string
	Boundary  string
	Chunked   bool
	Offline   bool
	Verbose   bool
}

type inp struct {
	Method    string
	URL       string
	BodyType  string
	StdinData []byte
}

// request only debug output of http.Request input parser.
/*type request struct {
	reqType int
}*/

func (d *dbug) toString() (string, error) {
	dbg := struct {
		opts `json:"Options"`
		inp  `json:"Input"`
	}{
		opts: opts{
			Scheme:    d.opts.Scheme(),
			JSON:      d.opts.JSON,
			Form:      d.opts.Form,
			Multipart: d.opts.Multipart,
			Raw:       d.opts.Raw,
			Boundary:  d.opts.Boundary,
			Chunked:   d.opts.Chunked,
			Offline:   d.opts.Offline,
			Verbose:   d.opts.Verbose,
		},
		inp: inp{
			Method:    d.inp.Method,
			URL:       d.inp.URL,
			BodyType:  d.inp.BodyType.String(),
			StdinData: d.inp.StdinData,
		},
	}
	debugData, err := json.MarshalIndent(dbg, "", ihttp.TabSpaces)
	if err != nil {
		return "", fmt.Errorf("debug print error: %v", err)
	}
	return string(debugData), nil
}
