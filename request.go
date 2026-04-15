package ihttp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type request struct {
	*http.Request
}

// NewRequest builds and start process to return HTTP Request from inp values.
func NewRequest(inp *Input) (*http.Request, error) {
	b, err := parseRequestBody(inp)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(inp.Method, inp.URL, bytes.NewBuffer(b.content))
	if err != nil {
		return nil, err
	}
	if req.Header.Get("Content-Type") == "" && b.contentType != "" {
		req.Header.Set("Content-Type", b.contentType)
	}
	r := &request{req}
	err = r.parseHeaders(inp)
	if err != nil {
		return nil, err
	}
	err = r.parseQuery(inp)
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

type itemValFunc func(item) string

// parseRequestBody parse Key and Val fields from inp.Items to objectJSON
// (that it could be later encode to JSON format data for the `body` argument
// to http.NewRequest).
//
// TODO: Try to do this with generics.
func parseRequestBody(inp *Input) (bodyTuple, error) {
	obj := make(objectJSON)
	var rules any = map[string]any{
		SepDataString: func() (itemValFunc, objectJSON) {
			return itemVal, obj
		},
	}
	for _, item := range inp.Items {
		if fn, ok := rules.(map[string]any)[item.Sep].(func() (itemValFunc, objectJSON)); ok {
			valFunc, targetMap := fn()
			value := valFunc(item)
			targetMap[item.Key] = value
		}
	}
	switch inp.BodyType {
	case EmptyBody:
		return bodyTuple{}, nil
	case JSONBody:
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
		for _, item := range inp.Items {
			formData.Add(item.Key, item.Val)
		}
		return bodyTuple{
			content:     []byte(formData.Encode()),
			contentType: "application/x-www-form-urlencoded",
		}, nil
	case RawBody:
		return bodyTuple{
			content:     inp.StdinData,
			contentType: "application/json",
		}, nil
	default:
		return bodyTuple{}, fmt.Errorf("unknown body type: %v", inp.BodyType)
	}
}

// itemVal is an itemValFunc type for the map `rules` in parseRequestBody.
func itemVal(i item) string { return i.Val }

// parseQuery add the Key and Val values from inp.Items to the URL Query string
// (type url.Values) of the HTTP Request using its Add method.
//
// TODO: Try to do this with generics.
func (r *request) parseQuery(inp *Input) (err error) {
	query := r.URL.Query()
	var rules any = map[string]any{
		SepQueryParam: func() (itemValFunc, url.Values) {
			return itemVal, query
		},
	}
	for _, i := range inp.Items {
		if fn, ok := rules.(map[string]any)[i.Sep].(func() (itemValFunc, url.Values)); ok {
			valFunc, targetMap := fn()
			value := valFunc(i)
			targetMap.Add(i.Key, value)
		}
	}
	r.URL.RawQuery = query.Encode()
	return nil
}

type itemValWithErrFunc func(item) (string, error)

// parseHeaders add the Key and Val values from inp.Items to Header of the HTTP
// Request using its Add, otherwise it will return error from some
// itemValWithErrFunc.
//
// TODO: Try to do this with generics.
func (r *request) parseHeaders(inp *Input) error {
	var rules any = map[string]any{
		SepHeader: func() (itemValWithErrFunc, http.Header) {
			return itemHeaderVal, r.Header
		},
		SepHeaderEmpty: func() (itemValWithErrFunc, http.Header) {
			return emptyHeaderVal, r.Header
		},
	}
	for _, i := range inp.Items {
		if fn, ok := rules.(map[string]any)[i.Sep].(func() (itemValWithErrFunc, http.Header)); ok {
			valFunc, targetMap := fn()
			value, err := valFunc(i)
			if err != nil {
				return err
			}
			targetMap.Add(i.Key, value)
		}
	}
	return nil
}

// headerVal is an itemValWithErrFunc type for the map `rules` in parseHeaders.
func itemHeaderVal(i item) (string, error) { return i.Val, nil }

// emptyHeaderVal is an itemValWithErrFunc type for the map `rules` in
// parseRequestHeaders.
func emptyHeaderVal(i item) (string, error) {
	if i.Val != "" {
		return i.Val, nil
	}
	return "", fmt.Errorf("invalid item %s (to specify an empty header use `Header;`)", i.Arg)
}
