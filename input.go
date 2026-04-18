package ihttp

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
)

type BodyType int

func (b BodyType) String() string {
	switch b {
	case EmptyBody:
		return "empty"
	case JSONBody:
		return "json"
	case FormBody:
		return "form"
	case MultipartBody:
		return "multipart"
	case RawBody:
		return "raw"
	default:
		return "unknown"
	}
}

const (
	EmptyBody     BodyType = iota
	JSONBody               // application/json
	FormBody               // application/x-www-form-urlencoded
	MultipartBody          // multipart/form-data
	RawBody                // application/octet-stream o detectado
)

var reMethod = regexp.MustCompile(`^[a-zA-Z]+$`)
var reScheme = regexp.MustCompile("^[a-z][a-z0-9.+-]*://")
var reShorthand = regexp.MustCompile(`^:(\d*)(\/?.*)$`)

// Input represent the input values from flags, cli args or stdin like pipelines.
type Input struct {
	Options   Options
	Method    string
	URL       string
	Items     []item
	StdinData []byte
	BodyType  BodyType
}

// NewInput return an Input pointer after parsing args o stdin value
// condicionated by the value flags from opts, otherwise return error.
func NewInput(args []string, opts Options) (*Input, error) {
	var method, url string
	var items []string
	switch len(args) {
	case 0:
		return nil, errors.New("URL is required")
	case 1:
		// Invoked as `$ http url`
		url = args[0]
	default:
		if reMethod.MatchString(args[0]) {
			// For example `$ http url foo=var field:value`
			method = args[0] // url
			url = args[1]    // foo=var
			items = args[2:] // field:value
		} else {
			// For example `$ http url/get field:value`
			url = args[0]    // url/get
			items = args[1:] // field:value
		}
	}
	if err := opts.IsValid(); err != nil {
		return nil, err
	}
	in := Input{Options: opts}
	// Set Items by parsing args to items.
	err := in.processItems(items)
	if err != nil {
		return nil, err
	}
	// Set StdinData via pipeline or -raw flag.
	err = in.processStdin(os.Stdin)
	if err != nil {
		return nil, err
	}
	// Set BodyType by StdinData or Items and options flags.
	in.processBodyType()
	// Set HTTP Method.
	err = in.processMethod(method)
	if err != nil {
		return nil, err
	}
	// Set URL.
	err = in.processURL(url)
	if err != nil {
		return nil, err
	}
	return &in, nil
}

// processItems parse each items as item struture.
func (in *Input) processItems(items []string) error {
	if len(items) == 0 {
		return nil
	}
	in.Items = nil
	seps := SepsGroupAllItems()
	for _, raw := range items {
		it, err := parseItem(raw, seps)
		if err != nil {
			return err
		}
		in.Items = append(in.Items, it)
	}
	return nil
}

// processBodyType set BodyType value by the priority of options flags
// and items separators.
func (in *Input) processBodyType() {
	// 1. Priority based on options flags
	if in.Options.Raw != "" || in.StdinData != nil {
		in.BodyType = RawBody
		return
	}
	if in.Options.Multipart {
		in.BodyType = MultipartBody
		return
	}
	if in.Options.Form {
		in.BodyType = FormBody
		return
	}
	// If the user uses -json, we force JSONBody
	if in.Options.JSON {
		in.BodyType = JSONBody
		return
	}
	// 2. Inference based on item separators
	var hasJSON, hasFile, hasData bool
	for _, it := range in.Items {
		switch it.Sep {
		case SepDataRawJSON:
			hasJSON = true
		case SepFileUpload:
			hasFile = true
		case SepDataString:
			hasData = true
		}
	}
	switch {
	case hasFile:
		in.BodyType = MultipartBody
	case hasJSON:
		in.BodyType = JSONBody
	case hasData:
		// If there are data items (key=val) but -form is not specified,
		// we assume JSON by default.
		in.BodyType = JSONBody
	default:
		// If there are no data items and no flags, the body will be empty.
		in.BodyType = EmptyBody
	}
}

// processStdin read the stdin data if exists and set BodyType to RawBody.
// Also check that only one of the data sources is used: items, -raw or stdin.
func (in *Input) processStdin(stdin io.Reader) error {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return err
	}
	hasStdin := (stat.Mode() & os.ModeCharDevice) == 0
	err = ensureOneDataSource(in.Items, in.Options, hasStdin)
	if err != nil {
		return err
	}
	// raw flag override
	if in.Options.Raw != "" {
		//in.BodyType = RawBody
		in.StdinData = []byte(in.Options.Raw)
		return nil
	}
	if hasStdin {
		//in.BodyType = RawBody
		in.StdinData, err = io.ReadAll(stdin)
		if err != nil {
			return err
		}
	}
	return nil
}

// ensureOneDataSource it can only be one source of input request data.
func ensureOneDataSource(items []item, opts Options, hasStdin bool) error {
	var hasDataItems bool
	dataSeps := SepsGroupDataItems()
	for _, it := range items {
		for _, sep := range dataSeps {
			if it.Sep == sep {
				hasDataItems = true
				break
			}
		}
		if hasDataItems {
			break
		}
	}
	hasRaw := opts.Raw != ""
	count := 0
	if hasDataItems {
		count++
	}
	if hasRaw {
		count++
	}
	if hasStdin {
		count++
	}
	if count > 1 {
		return errors.New("request body (stdin, -raw, or file) and request data items (key=value, key:=value) cannot be mixed")
	}
	return nil
}

// processMethod set HTTP Method.
func (in *Input) processMethod(method string) error {
	if method != "" {
		if !reMethod.MatchString(method) {
			return fmt.Errorf("METHOD must consist of alphabet: %s", method)
		}
		in.Method = strings.ToUpper(method)
	} else {
		in.Method = guessMethod(in.BodyType)
	}
	return nil
}

// guessMethod return the default HTTP Method by BodyType value.
func guessMethod(bodyType BodyType) string {
	if bodyType == EmptyBody {
		return http.MethodGet
	} else {
		return http.MethodPost
	}
}

// processURL set URL value, works as parseURL in httpie-go.
func (in *Input) processURL(url string) error {
	// Prepare the url for add the scheme: `http ://domain.com` → `http://domain.com`.
	url = strings.TrimPrefix(url, "://")
	// Check scheme: if the URL doesn't specify the protocol,
	// then precede it with http:// or https://
	if !reScheme.MatchString(url) {
		scheme := in.Options.Scheme() + "://"
		if in.Options.Scheme() == "https" {
			scheme = "https://"
		}
		// See if we're using curl style shorthand for localhost, e.g.
		// :3000/foo, if is success it will be return a slice with following elements:
		//
		//	matches[0] :3000/foo
		//	matches[1] :3000
		//	matches[2] foo
		sh := reShorthand.FindStringSubmatch(url)
		if len(sh) == 3 {
			port := sh[1]
			rest := sh[2]
			if strings.HasPrefix(url, "::") {
				url = scheme + ":"
			} else {
				url = scheme + "localhost"
			}
			if port != "" {
				url += ":" + port
			}
			url += rest
		} else {
			url = scheme + url
		}
	}
	in.URL = url
	return nil
}
