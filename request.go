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
	var root any

	for _, item := range items {
		// Only consider items that are meant for the body (data or raw JSON),
		// skip others like headers or query params
		if item.Sep != SepDataString && item.Sep != SepDataRawJSON {
			continue
		}

		// Parsing path from the key, e.g., "a[b][c]" to ["a", "b", "c"].
		path, err := parseKey(item.Key)
		if err != nil {
			return bodyTuple{}, err
		}

		// Set values, for string data, we can use it directly.
		// For raw JSON, we need to unmarshal it, etc.
		var v any
		switch item.Sep {
		case SepDataString:
			v = item.Val

		case SepDataRawJSON:
			if err := json.Unmarshal([]byte(item.Val), &v); err != nil {
				return bodyTuple{}, fmt.Errorf("invalid JSON value for %q: %w", item.Key, err)
			}
		}

		// Insert the value into the root object at the specified path.
		root, err = insertJSON(root, path, v)
		if err != nil {
			return bodyTuple{}, err
		}
	}

	// Edge case: without items {} or only non-data items,
	// we should return an empty JSON object:
	if root == nil {
		root = map[string]any{}
	}

	b, err := json.Marshal(root)
	if err != nil {
		return bodyTuple{}, err
	}
	return bodyTuple{
		content:     b,
		contentType: "application/json",
	}, nil
}

// pathSegment represents one parsed segment of a bracket-notation key.
// For example, value "a[0][b]" would be parsed into segments: "a", "0", and "b".
// The "escaped" field indicates whether the segment was escaped with a backslash,
// which affects how it should be treated (e.g., as a literal key rather than an
// array index).
type pathSegment struct {
	value   string
	escaped bool // true when the value was preceded by \ (treat int as string key)
}

// parseKey parses a key with bracket notation into a slice of path segments.
// For example, "foo[bar][baz]" would be parsed into ["foo", "bar", "baz"].
func parseKey(k string) ([]pathSegment, error) {
	var parts []pathSegment
	var buf strings.Builder
	var nextEscaped bool
	inBracket := false

	for i := 0; i < len(k); i++ {
		ch := k[i]
		switch ch {
		case '\\':
			if i+1 >= len(k) {
				return nil, fmt.Errorf("invalid escape at end of %q", k)
			}
			i++
			buf.WriteByte(k[i])
			nextEscaped = true

		case '[':
			if inBracket {
				return nil, fmt.Errorf("unexpected '[' in %q", k)
			}
			if buf.Len() > 0 {
				parts = append(parts, pathSegment{value: buf.String(), escaped: nextEscaped})
				buf.Reset()
				nextEscaped = false
			}
			inBracket = true

		case ']':
			if !inBracket {
				return nil, fmt.Errorf("unexpected ']' in %q", k)
			}
			parts = append(parts, pathSegment{value: buf.String(), escaped: nextEscaped})
			buf.Reset()
			nextEscaped = false
			inBracket = false

		default:
			buf.WriteByte(ch)
		}
	}

	if inBracket {
		return nil, fmt.Errorf("missing ']' in %q", k)
	}
	if buf.Len() > 0 {
		parts = append(parts, pathSegment{value: buf.String(), escaped: nextEscaped})
	}
	return parts, nil
}

// insertJSON inserts a value into a nested map or slice structure based on the
// provided path.
func insertJSON(current any, path []pathSegment, value any) (any, error) {
	if len(path) == 0 {
		return value, nil
	}
	seg := path[0]
	rest := path[1:]

	// Treat as map key when: not an index pattern, OR explicitly escaped.
	if !isIndex(seg.value) || seg.escaped {
		var obj map[string]any
		if current == nil {
			obj = map[string]any{}
		} else {
			var ok bool
			obj, ok = current.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("type error: expected object at %q", seg.value)
			}
		}
		updated, err := insertJSON(obj[seg.value], rest, value)
		if err != nil {
			return nil, err
		}
		obj[seg.value] = updated
		return obj, nil
	}

	// ARRAY
	var arr []any
	if current == nil {
		arr = []any{}
	} else {
		var ok bool
		arr, ok = current.([]any)
		if !ok {
			return nil, fmt.Errorf("type error: expected array at %q", seg.value)
		}
	}

	// append []
	if seg.value == "" {
		if len(rest) == 0 {
			return append(arr, value), nil
		}
		newItem, err := insertJSON(nil, rest, value)
		if err != nil {
			return nil, err
		}
		return append(arr, newItem), nil
	}

	// numeric index
	idx, _ := strconv.Atoi(seg.value)
	for len(arr) <= idx {
		arr = append(arr, nil)
	}
	updated, err := insertJSON(arr[idx], rest, value)
	if err != nil {
		return nil, err
	}
	arr[idx] = updated
	return arr, nil
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

// buildMultipartBody constructs the body content and content type for a multipart
// body based.
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
