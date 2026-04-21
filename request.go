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
)

type request struct {
	*http.Request
}

// NewRequest builds and start process to return HTTP Request from inp values.
func NewRequest(in *Input) (*http.Request, error) {
	b, err := parseRequestBody(in)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(in.Method, in.URL, bytes.NewBuffer(b.content))
	if err != nil {
		return nil, err
	}
	if req.Header.Get("Content-Type") == "" && b.contentType != "" {
		req.Header.Set("Content-Type", b.contentType)
	}
	r := &request{req}
	err = r.parseHeaders(in)
	if err != nil {
		return nil, err
	}
	err = r.parseQuery(in)
	if err != nil {
		return nil, err
	}
	return r.Request, nil
}

type bodyTuple struct {
	content     []byte
	contentType string
}

// objectJSON represent a JSON object as a map. You can get it as JSON-encoded data.
type objectJSON map[string]any

// toData execute internally json.Marshal for get it as JSON-encoded data.
// If the JSON object map is empty, it will return nil as zero value of []byte.
func (oj objectJSON) toData() (data []byte, err error) {
	if len(oj) > 0 {
		data, err = json.Marshal(oj)
		if err != nil {
			return nil, err
		}
	}
	return data, nil
}

// parseRequestBody parse Key and Val fields from inp.Items to objectJSON
// (that it could be later encode to JSON format data for the `body` argument
// to http.NewRequest).
func parseRequestBody(in *Input) (bodyTuple, error) {
	switch in.BodyType {
	case EmptyBody:
		return bodyTuple{}, nil
	case JSONBody:
		obj := make(objectJSON)
		for _, it := range in.Items {
			if it.Sep == SepDataString {
				obj[it.Key] = it.Val
			}
		}
		bodyData, err := obj.toData()
		if err != nil {
			return bodyTuple{}, fmt.Errorf("marshaling JSON of HTTP body: %v", err)
		}
		return bodyTuple{
			content:     bodyData,
			contentType: "application/json",
		}, nil
	case FormBody:
		formData := url.Values{}
		for _, item := range in.Items {
			formData.Add(item.Key, item.Val)
		}
		return bodyTuple{
			content:     []byte(formData.Encode()),
			contentType: "application/x-www-form-urlencoded",
		}, nil
	case MultipartBody:
		return buildMultipartBody(in)
	case RawBody:
		return bodyTuple{
			content:     in.StdinData,
			contentType: "application/json",
		}, nil
	default:
		return bodyTuple{}, fmt.Errorf("unknown body type: %v", in.BodyType)
	}
}

func buildMultipartBody(in *Input) (bodyTuple, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	// when -boundary flag is set
	if in.Options.Boundary != "" {
		writer.SetBoundary(in.Options.Boundary)
	}
	for _, it := range in.Items {
		switch it.Sep {
		case SepFileUpload:
			file, err := os.Open(it.Val)
			if err != nil {
				return bodyTuple{}, err
			}
			defer file.Close()
			part, err := writer.CreateFormFile(it.Key, it.Val)
			if err != nil {
				return bodyTuple{}, err
			}
			_, err = io.Copy(part, file)
			if err != nil {
				return bodyTuple{}, err
			}
		case SepDataString:
			err := writer.WriteField(it.Key, it.Val)
			if err != nil {
				return bodyTuple{}, err
			}
		}
	}
	writer.Close()
	return bodyTuple{
		content:     buf.Bytes(),
		contentType: writer.FormDataContentType(),
	}, nil
}

// parseQuery add the Key and Val values from inp.Items to the URL Query string
// (type url.Values) of the HTTP Request using its Add method.
func (r *request) parseQuery(in *Input) (err error) {
	query := r.URL.Query()
	r.URL.RawQuery = query.Encode()
	return nil
}

// parseHeaders add the Key and Val values from inp.Items to Header of the HTTP
// Request using its Add, otherwise it will return error from some
// itemValWithErrFunc.
func (r *request) parseHeaders(in *Input) error {
	for _, i := range in.Items {
		switch i.Sep {
		case SepHeader:
			r.Header.Add(i.Key, i.Val)
		case SepHeaderEmpty:
			if i.Val == "" {
				return fmt.Errorf("invalid item %s (to specify an empty header use `Header;`)", i.Arg)
			}
			r.Header.Add(i.Key, i.Val)
		}
	}
	return nil
}
