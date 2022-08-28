package ihttp

import (
	"fmt"
	"sort"
	"strings"
)

// Item a key-value pair struct for the arguments considered as ITEMS after
// the URL.
type item struct {
	// Key represent the value string before of a separator.
	Key string

	// Val represent the value string after of a separator.
	Val string

	// Sep is the separator of an argument in the CLI.
	Sep string

	// Arg represent a fully argument from the CLI after the URL.
	Arg string
}

// parseItem parse a raw string argument that it contains a separator
// represented in the group of seperators (seps): headers, form data
// (body request) and other key-value pair types.
//
// The back slash escaped characters aren't considered as seps (or parts
// thereof). Literal back slash characters have to be escaped as well (`\\`).
//
// parseItem is used internally to parse items in Parser.
func parseItem(arg string, seps []string) (item, error) {
	var sep, key, value string

	tokens := tokenize(arg, seps)

	// Sort by lenght ensures that the longest one will be
	// chosen as it will overwrite any shorter ones starting
	// at the same position in the `found` map.
	sort.Slice(seps, func(i, j int) bool {
		return len(seps[i]) < len(seps[j])
	})

	if func() bool {
		for i, token := range tokens {
			if escaped, ok := token.([]byte); ok {
				tokens[i] = string(escaped)
				continue
			}

			found := make(find)

			for _, sep := range seps {
				pos := strings.Index(token.(string), sep)
				if pos != -1 {
					found[pos] = sep
				}
			}

			if len(found) > 0 {
				// Starting first, longest separator found.
				sep = found[found.min()]

				r := strings.SplitN(token.(string), sep, 2)
				key, value = r[0], r[1]

				// Any preceding tokens are part of the key.
				key = strings.Join(toStrSlice(tokens[:i]), "") + key

				// Any following tokens are part of the value.
				value += strings.Join(toStrSlice(tokens[i+1:]), "")

				return false
			}
		}
		return true
	}() {
		return item{}, fmt.Errorf("%s is not a valid value", arg)
	}

	return item{
		Key: key,
		Val: value,
		Sep: sep,
		Arg: arg,
	}, nil
}

// tokenize tokenize the raw arg string. There are only two token types,
// strings and escaped characters, usage example:
//
//     tokenize(`foo\=bar\\baz`, []string{"="}):
//
// Result:
//
//     [foo = bar\\baz]
func tokenize(arg string, seps []string) []interface{} {
	var tokens = []interface{}{""}
	var i int

	for i < len(arg) {
		if char := string(arg[i]); char == "\\" {

			if i++; i < len(arg) {
				char = string(arg[i])
			} else {
				char = "" // Default char of next func ;)
			}

			if !inStrSlice(char, seps) {
				s := tokens[len(tokens)-1].(string)
				s += "\\" + char
				tokens[len(tokens)-1] = s
			} else {
				// Catch escaped character by conversion to []byte.
				tokens = append(tokens, []byte(char), "")
			}

		} else {
			s := tokens[len(tokens)-1].(string)
			s += char
			tokens[len(tokens)-1] = s
		}

		i++
	}
	return tokens
}

// find map for mark seperators found tokenize algorithm.
type find map[int]string

// min returns smallest key.
func (f find) min() int {
	// Make integer list of found map keys, where they will be stored.
	keys := make([]int, 0, len(f))
	for k := range f {
		keys = append(keys, k)
	}

	// Set the smallest number to the first element of the list.
	smallest := keys[0]
	for _, key := range keys[1:] {
		if key < smallest {
			smallest = key
		}
	}
	return smallest
}

// toStrSlice return elements from i in a string slice.
func toStrSlice(i []interface{}) []string {
	ss := make([]string, len(i))
	for k, v := range i {
		if s, ok := v.(string); ok {
			ss[k] = s
		}
	}
	return ss
}

// inStrSlice return true if str is present in slice.
func inStrSlice(str string, slice []string) bool {
	for _, v := range slice {
		if v == str {
			return true
		}
	}
	return false
}
