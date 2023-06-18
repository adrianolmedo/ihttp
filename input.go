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

type bodyType int

const (
	emptyBody bodyType = iota
	jsonBody
	formBody
	rawBody
)

var reMethod = regexp.MustCompile(`^[a-zA-Z]+$`)

// Input represent the input values from flags, cli args or stdin like pipelines.
type Input struct {
	Options   Options
	Method    string
	URL       string
	Items     []item // Used when BodyType is JSONBody or FormBody.
	StdinData []byte
	BodyType  bodyType
}

// NewInput return an Input pointer after parsing args o stdin value
// condicionated by the value flags from opts, otherwise return error.
func NewInput(args []string, stdin io.Reader, opts ...Options) (*Input, error) {
	var method, url string
	var items []string

	switch len(args) {
	case 0:
		return nil, errors.New("URL is required")
	case 1:
		// Invoked as `$ http url`:
		url = args[0]
	default:
		if reMethod.MatchString(args[0]) {
			// For example `$ http url foo=var field:value`:
			method = args[0] // url
			url = args[1]    // foo=var
			items = args[2:] // field:value
		} else {
			// For example `$ http url/get field:value`:
			url = args[0]    // url/get
			items = args[1:] // field:value
		}
	}

	var inp Input
	for _, o := range opts {
		if err := o.IsValid(); err != nil {
			return nil, err
		}

		inp = Input{Options: o}
	}

	err := inp.processItems(items)
	if err != nil {
		return nil, err
	}

	err = inp.processStdin(stdin)
	if err != nil {
		return nil, err
	}

	err = inp.processMethod(method)
	if err != nil {
		return nil, err
	}

	err = inp.processURL(url)
	if err != nil {
		return nil, err
	}

	return &inp, nil
}

// getBodyType works as determinePreferredBodyType in httpie-go and estimate
// the bodyType from opts values.
func getBodyType(opts Options) bodyType {
	if opts.Form {
		return formBody
	} else {
		return jsonBody
	}
}

// processItems parse each items as item struture.
func (inp *Input) processItems(items []string) (err error) {
	if len(items) >= 1 {
		// BUG: When pass the flag -form the inp.BodyType is not setting to JSONBody.
		bodyType := getBodyType(inp.Options)

		seps := SepsGroupAllItems()
		inp.Items = make([]item, len(items))

		for i := 0; i < len(items); i++ {
			// parseItem works as splitItem in httpie-go.
			// inp.Items[i] = Item, works as parseField in httpie-go.
			inp.Items[i], err = parseItem(items[i], seps)
			if err != nil {
				return err
			}

			if inp.Items[i].Sep == SepDataString {
				inp.BodyType = bodyType
			}

			if inp.Items[i].Sep == SepDataRawJSON {
				inp.BodyType = jsonBody
			}

			if inp.Items[i].Sep == SepFileUpload {
				inp.BodyType = formBody
			}
		}
	}

	return nil
}

func (inp *Input) processStdin(stdin io.Reader) error {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return err
	}

	// The next line works as options.ReadStdin && !state.stdinConsume in httpie-go.
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		// These two approaches for specifying request item (i.e., structured and raw) cannot be combined.
		if inp.BodyType != emptyBody {
			return errors.New("request body (from stdin) and request item (key=value) cannot be mixed")
		}

		inp.BodyType = rawBody
		inp.StdinData, err = io.ReadAll(stdin)
		if err != nil {
			return err
		}
	}

	return nil
}

// processMethod set HTTP Method.
func (inp *Input) processMethod(method string) error {
	if method != "" {
		if !reMethod.MatchString(method) {
			return fmt.Errorf("METHOD must consist of alphabet: %s", method)
		}

		inp.Method = strings.ToUpper(method)
	} else {
		inp.Method = guessMethod(inp.BodyType)
	}
	return nil
}

func guessMethod(bodyType bodyType) string {
	if bodyType == emptyBody {
		return http.MethodGet
	} else {
		return http.MethodPost
	}
}

// processURL set URL value, works as parseURL in httpie-go.
func (inp *Input) processURL(url string) error {
	// Prepare the url for add the scheme: `http ://domain.com` → `http://domain.com`.
	url = strings.TrimPrefix(url, "://")

	reScheme, err := regexp.Compile("^[a-z][a-z0-9.+-]*://")
	if err != nil {
		return err
	}

	// Check scheme: if the URL doesn't specify the protocol,
	// then precede it with http:// or https://
	if !reScheme.MatchString(url) {
		scheme := inp.Options.Scheme() + "://"
		if inp.Options.Scheme() == "https" {
			scheme = "https://"
		}

		sh, err := getShorthand(url)
		if err != nil {
			return err
		}

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

	inp.URL = url
	return nil
}

// getShorthand see if we're using curl style shorthand for localhost, e.g.
// :3000/foo, if is success it will be return a slice with following elements:
//
//	matches[0] :3000/foo
//	matches[1] :3000
//	matches[2] foo
//
// Or nil and err otherwise.
func getShorthand(url string) (matches []string, err error) {
	rgx, err := regexp.Compile(`^:(\d*)(\/?.*)$`)
	if err != nil {
		return nil, err
	}
	return rgx.FindStringSubmatch(url), nil
}
