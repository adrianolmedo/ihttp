package ihttp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type request struct {
	*http.Request
}

// NewRequest builds and start process to return HTTP Request from [Input] values.
func NewRequest(in *Input) (*http.Request, []byte, error) {
	b, err := buildBody(in)
	if err != nil {
		return nil, nil, err
	}
	var bodyReader io.Reader = http.NoBody
	if len(b.content) > 0 {
		bodyReader = bytes.NewBuffer(b.content)
	}
	req, err := http.NewRequest(in.Method, in.URL, bodyReader)
	if err != nil {
		return nil, nil, err
	}
	if req.Header.Get("Content-Type") == "" && b.contentType != "" {
		req.Header.Set("Content-Type", b.contentType)
	}
	r := &request{req}
	err = r.buildHeaders(in)
	if err != nil {
		return nil, nil, err
	}
	r.buildDefaultHeaders(in)
	err = r.buildURLQuery(in)
	if err != nil {
		return nil, nil, err
	}
	return r.Request, b.content, nil
}

type bodyTuple struct {
	content     []byte
	contentType string
}

func buildBody(in *Input) (bodyTuple, error) {
	switch in.BodyType {
	case EmptyBody:
		return bodyTuple{}, nil

	case RawBody:
		return bodyTuple{
			content:     in.StdinData,
			contentType: "application/json",
		}, nil

	case JSONBody:
		return buildJSONBody(in.Items)

	case FormBody:
		return buildFormBody(in.Items)

	case MultipartBody:
		return buildMultipartBody(in.Items, in.Options.Boundary)

	default:
		return bodyTuple{}, fmt.Errorf("unsupported body type: %s", in.BodyType)
	}
}

// buildJSONBody constructs the body content and content type for a JSON body based.
func buildJSONBody(items []item) (bodyTuple, error) {
	data := make(map[string]any)
	for _, it := range items {
		switch it.Sep {
		case SepDataString:
			data[it.Key] = it.Val
		case SepDataRawJSON:
			var v any
			if err := json.Unmarshal([]byte(it.Val), &v); err != nil {
				return bodyTuple{}, fmt.Errorf("invalid JSON value for %q: %w", it.Key, err)
			}
			data[it.Key] = v
		}
	}
	b, err := json.Marshal(data)
	if err != nil {
		return bodyTuple{}, err
	}
	return bodyTuple{content: b, contentType: "application/json"}, nil
}

// buildFormBody constructs the body content and content type for a form body based.
func buildFormBody(items []item) (bodyTuple, error) {
	// if any file fields are present, delegate to multipart
	for _, it := range items {
		if it.Sep == SepFileUpload {
			return buildMultipartBody(items, "")
		}
	}
	vals := url.Values{}
	for _, it := range items {
		if it.Sep == SepDataString {
			vals.Add(it.Key, it.Val)
		}
	}
	return bodyTuple{
		content:     []byte(vals.Encode()),
		contentType: "application/x-www-form-urlencoded; charset=utf-8",
	}, nil
}

// buildMultipartBody constructs the body content and content type for a multipart body based.
func buildMultipartBody(items []item, boundary string) (bodyTuple, error) {
	var buf bytes.Buffer
	var w *multipart.Writer
	if boundary != "" {
		w = multipart.NewWriter(&buf)
		w.SetBoundary(boundary)
	} else {
		w = multipart.NewWriter(&buf)
	}
	for _, it := range items {
		switch it.Sep {
		case SepDataString:
			if err := w.WriteField(it.Key, it.Val); err != nil {
				return bodyTuple{}, err
			}
		case SepFileUpload:
			f, err := os.Open(it.Val)
			if err != nil {
				return bodyTuple{}, fmt.Errorf("cannot open file %q: %w", it.Val, err)
			}
			part, err := w.CreateFormFile(it.Key, filepath.Base(it.Val))
			if err != nil {
				f.Close()
				return bodyTuple{}, err
			}
			if _, err := io.Copy(part, f); err != nil {
				f.Close()
				return bodyTuple{}, err
			}
			f.Close()
		}
	}
	if err := w.Close(); err != nil {
		return bodyTuple{}, err
	}
	return bodyTuple{content: buf.Bytes(), contentType: w.FormDataContentType()}, nil
}

// buildURLQuery add the Key and Val values from in.Items to the URL Query string
// (type url.Values) of the HTTP Request using its Add method.
func (r *request) buildURLQuery(in *Input) error {
	query := r.URL.Query()
	for _, it := range in.Items {
		if it.Sep == SepQueryParam {
			query.Add(it.Key, it.Val)
		}
	}
	r.URL.RawQuery = query.Encode()
	return nil
}

// buildHeaders add the Key and Val values from in.Items to Header of the HTTP
// Request using its Add, otherwise it will return error.
func (r *request) buildHeaders(in *Input) error {
	for _, i := range in.Items {
		switch i.Sep {
		case SepHeader:
			if strings.EqualFold(i.Key, "Host") {
				r.Host = i.Val // overrides the Host header specifically
			} else {
				r.Header.Add(i.Key, i.Val)
			}
		case SepHeaderEmpty:
			if i.Val == "" {
				return fmt.Errorf("invalid item %s (to specify an empty header use `Header;`)", i.Arg)
			}
			r.Header.Add(i.Key, i.Val)
		}
	}
	return nil
}

// buildDefaultHeaders sets default headers.
func (r *request) buildDefaultHeaders(in *Input) {
	if in.BodyType == JSONBody {
		if r.Header.Get("Accept") == "" {
			r.Header.Set("Accept", "application/json, */*;q=0.5")
		}
		if r.Header.Get("Content-Type") == "" {
			// already set from buildBody, but guard anyway
			r.Header.Set("Content-Type", "application/json")
		}
	}
}
