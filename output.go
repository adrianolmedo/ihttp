package ihttp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httputil"
	"sort"
	"strings"
	"time"
)

type Output struct {
	Request *http.Request
	Options Options

	sb  strings.Builder
	err error
}

func NewOutput(req *http.Request, opts Options) (*Output, error) {
	o := &Output{Request: req, Options: opts}

	if o.Options.Verbose {
		o.writeRequest()
	}

	o.writeResponse()

	if o.err != nil {
		return nil, o.err
	}
	return o, nil
}

// withErr filters the contents of the Output render through the supplied
// function, which returns an error, which will be set on Output.
func (o *Output) withErr(filter func() error) {
	o.err = filter()
}

// writeHeaders write Headers from h.
func (o *Output) writeHeaders(h http.Header) {
	for _, vs := range sortHeaderKeys(h) {
		o.sb.WriteString(vs + ": ")
		for _, v := range h[vs] {
			o.sb.WriteString(v + "\n")
		}
	}
}

// sortHeaderKeys sort alphabetically h and return it as string slice.
func sortHeaderKeys(h http.Header) []string {
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// writeRequest write the HTTP Request from Request parsed.
func (o *Output) writeRequest() {
	o.withErr(func() error {
		reqData, err := httputil.DumpRequestOut(o.Request, true)
		if err != nil {
			return err
		}

		br := bufio.NewReader(bytes.NewReader(reqData))
		r, err := http.ReadRequest(br)
		if err != nil {
			return err
		}

		o.sb.WriteString(r.Method + " " + r.URL.Path + " " + r.Proto + "\n")
		if len(r.Header.Values("host")) == 0 && o.Request.URL.Host != "" {
			r.Header.Add("Host", o.Request.URL.Host)
		}

		o.writeHeaders(r.Header)
		o.writeRequestBody(r)
		o.sb.WriteString("\n")
		return nil
	})
}

// writeRequestBody write the Body from r.
func (o *Output) writeRequestBody(r *http.Request) {
	o.withErr(func() error {
		defer r.Body.Close()

		bodyData, err := io.ReadAll(r.Body)
		if err != nil {
			return err
		}

		var body string
		if isJSON(bytes.NewBuffer(bodyData)) {
			bodyBuf := &bytes.Buffer{}

			if err := json.Indent(bodyBuf, bodyData, "", TabSpaces); err != nil {
				return err
			}

			body = bodyBuf.String()
		} else {
			body = string(bodyData)
		}

		o.sb.WriteString("\n" + body)
		return nil
	})
}

// isJSON returns true if the streamed input r is a valid JSON format.
func isJSON(r io.Reader) bool {
	dec := json.NewDecoder(r)
	for {
		_, err := dec.Token()
		if err == io.EOF {
			//break
			return true // end of input, valid JSON
		}

		if err != nil {
			return false // invalid JSON
		}
	}
}

// writeResponse make an HTTP Response from Request parsed and then render
// to string.
func (o *Output) writeResponse() {
	o.withErr(func() error {
		r, err := newResponse(o.Request)
		if err != nil {
			return err
		}

		o.sb.WriteString(r.Proto + " " + r.Status + "\n")
		o.writeHeaders(r.Header)
		o.writeResponseBody(r)
		return nil
	})
}

// newResponse helper that returns a *http.Response given a *http.Request.
func newResponse(req *http.Request) (*http.Response, error) {
	client := &http.Client{
		Timeout: time.Second * 30,
	}

	return client.Do(req)
}

// writeResponseBody write the Body from r.
func (o *Output) writeResponseBody(r *http.Response) {
	o.withErr(func() error {
		defer r.Body.Close()

		bodyData, err := io.ReadAll(r.Body)
		if err != nil {
			return err
		}

		var body string
		if strings.Contains(r.Header.Get("content-type"), "application/json") && isJSON(r.Body) {
			bodyBuf := &bytes.Buffer{}
			if err := json.Indent(bodyBuf, bodyData, "", TabSpaces); err != nil {
				return err
			}

			body = bodyBuf.String()
		} else {
			body = string(bodyData)
		}

		o.sb.WriteString("\n" + body)
		return nil
	})
}

// String return the HTTP Response and depending on the Options values it will
// also include debug or HTTP Request output if Options.Verbose is true.
func (o *Output) String() string {
	return o.sb.String()
}
