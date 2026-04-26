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
	"strconv"
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
	for _, item := range items {
		switch item.Sep {

		case SepDataString:
			path := parseKey(item.Key)
			insertJSON(data, path, item.Val)

		case SepDataRawJSON:
			path := parseKey(item.Key)
			var v any
			if err := json.Unmarshal([]byte(item.Val), &v); err != nil {
				return bodyTuple{}, fmt.Errorf("invalid JSON value for %q: %w", item.Key, err)
			}
			insertJSON(data, path, v)
		}
	}
	b, err := json.Marshal(data)
	if err != nil {
		return bodyTuple{}, err
	}
	return bodyTuple{content: b, contentType: "application/json"}, nil
}

// parseKey splits a key like "a[b][c]" into its parts: ["a", "b", "c"].
func parseKey(k string) []string {
	var parts []string
	buf := ""
	for i := 0; i < len(k); i++ {
		switch k[i] {
		case '[':
			if buf != "" {
				parts = append(parts, buf)
				buf = ""
			}
		case ']':
			parts = append(parts, buf)
			buf = ""
		default:
			buf += string(k[i])
		}
	}
	if buf != "" {
		parts = append(parts, buf)
	}
	return parts
}

// insertJSON inserts a value into a nested map or slice structure based on the
// provided path.
func insertJSON(root map[string]any, path []string, value any) {
	var current any = root
	for i, p := range path {
		last := i == len(path)-1
		switch node := current.(type) {
		case map[string]any:
			if last {
				if existing, ok := node[p]; ok {
					node[p] = mergeValue(existing, value)
				} else {
					node[p] = value
				}
				return
			}
			next, exists := node[p]
			if !exists {
				if isIndex(path[i+1]) {
					next = []any{}
				} else {
					next = map[string]any{}
				}
				node[p] = next
			}
			current = next

		case []any:
			if p == "" {

				// append []
				if last {
					node = append(node, value)
					return
				}
				newMap := map[string]any{}
				node = append(node, newMap)
				current = newMap
				continue
			}
			idx, err := strconv.Atoi(p)
			if err != nil {
				panic("invalid index in path")
			}

			// expand
			for len(node) <= idx {
				node = append(node, nil)
			}
			if last {
				if node[idx] != nil {
					node[idx] = mergeValue(node[idx], value)
				} else {
					node[idx] = value
				}
				return
			}
			if node[idx] == nil {
				if isIndex(path[i+1]) {
					node[idx] = []any{}
				} else {
					node[idx] = map[string]any{}
				}
			}
			current = node[idx]
		}
	}
}

// mergeValue merges an existing value with an incoming value, combining them
// into a slice if necessary.
func mergeValue(existing, incoming any) any {
	switch e := existing.(type) {
	case []any:
		return append(e, incoming)
	case map[string]any:
		if m2, ok := incoming.(map[string]any); ok {
			for k, v := range m2 {
				if old, exists := e[k]; exists {
					e[k] = mergeValue(old, v)
				} else {
					e[k] = v
				}
			}
			return e
		}
		return []any{e, incoming}
	default:
		return []any{e, incoming}
	}
}

// isIndex checks if a string is an integer index (e.g., "0", "1", etc.)
// or empty (for appending to arrays).
func isIndex(s string) bool {
	if s == "" {
		return true
	}
	_, err := strconv.Atoi(s)
	return err == nil
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
