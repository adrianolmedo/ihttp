package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/adrianolmedo/ihttp"
)

var usage = `Usage:

    http [METHOD] URL [ITEMS ...]

Options:

    -scheme 	The default scheme to use if not specified in the URL. 
            	And if you set it empty, the value it will be 'http'.

    -json   	(default) Data items from the command line are serialized as a JSON object.
            	The Content-Type and Accept headers are set to application/json
            	(if not specified).

    -form   	Data items from the command line are serialized as form fields.
      
            	The Content-Type is set to application/x-www-form-urlencoded (if not
            	specified). The presence of any file fields results in a
            	multipart/form-data request.
				
    -multipart  Similar to --form, but always sends a multipart/form-data request
            	(i.e., even without files).
				
    -boundary  	Set the boundary parameter for multipart/form-data requests. 
            	This option is only relevant when using --multipart.
				
    -raw    	This option allows you to pass raw request data without extra processing
            	(as opposed to the structured request items syntax):
      
            		$ http --raw='data' httpbingo.org/post 
      
            	You can achieve the same by piping the data via stdin:
      
            		$ echo data | http httpbingo.org/post
      
            	Or have HTTPie load the raw data from a file:
      
            		$ http httpbingo.org/post @data.txt

    -v      	Verbose output. Print the whole request as well as the response.

    -debug  	Debug print info about iHTTP for debugging itself and for reporting bugs.

`

func main() {
	var (
		scheme    = flag.String("scheme", filepath.Base(os.Args[0]), "")
		json      = flag.Bool("json", false, "")
		form      = flag.Bool("form", false, "")
		multipart = flag.Bool("multipart", false, "")
		boundary  = flag.String("boundary", "", "")
		raw       = flag.String("raw", "", "")
		verbose   = flag.Bool("v", false, "")
		debug     = flag.Bool("debug", false, "")
	)
	// Set usage:
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, usage)
	}
	flag.Parse()
	// Set Option values from the flags:
	opts := ihttp.Options{
		JSON:      *json,
		Form:      *form,
		Multipart: *multipart,
		Boundary:  *boundary,
		Raw:       *raw,
		Verbose:   *verbose,
	}
	opts.SetScheme(*scheme)
	// Parse args to Input values.
	inp, err := ihttp.NewInput(flag.Args(), os.Stdin, opts)
	if err != nil {
		errAndExit(err)
	}
	// Optional show debug info.
	if *debug {
		dbug := &dbug{opts: opts, inp: inp}
		dbg, err := dbug.toString()
		if err != nil {
			errAndExit(err)
		}
		dbg = fmt.Sprintf("iHTTP v%s\n\n%s\n\n", ihttp.Version, dbg)
		fmt.Fprint(os.Stdout, dbg)
	}
	req, err := ihttp.NewRequest(inp)
	if err != nil {
		errAndExit(err)
	}
	out, err := ihttp.NewOutput(req, opts)
	if err != nil {
		errAndExit(err)
	}
	fmt.Fprint(os.Stdout, out)
}

/*func usageAndExit(msg string) {
	flag.Usage()
	if msg != "" {
		fmt.Fprintln(os.Stderr, "\nerror:", msg)
	}
	os.Exit(1)
}*/

func errAndExit(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
