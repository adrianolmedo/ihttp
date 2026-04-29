package ihttp

import (
	"fmt"
	"slices"
	"strings"
)

// item a key-value pair struct for the arguments considered as ITEMS after
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

// token it represents a piece of the original string.
type token struct {
	value   string
	escaped bool
}

// parseItem parse a raw string that it contains a separator represented in the
// group of seperators (seps): headers, form data (body request) and other
// key-value pair types.
//
// The back slash escaped characters aren't considered as seps (or parts
// thereof). Literal back slash characters have to be escaped as well (`\\`).
//
// parseItem is used internally to parse items in [Input.processItems] method.
func parseItem(arg string, seps []string) (item, error) {
	tokens := tokenize(arg, seps)
	for i, t := range tokens {
		if t.escaped {
			continue
		}
		minPos := -1

		// chosenSep is the separator that is found at the minPos.
		// We need to keep track of it to split the token value correctly.
		var chosenSep string
		for _, s := range seps {
			pos := strings.Index(t.value, s)
			if pos != -1 {
				if minPos == -1 ||
					pos < minPos ||
					(pos == minPos && len(s) > len(chosenSep)) {
					minPos = pos
					chosenSep = s
				}
			}
		}
		if minPos != -1 {
			r := strings.SplitN(t.value, chosenSep, 2)
			keyLeft, valueRight := r[0], r[1]

			// Rebuild the key and value by concatenating the tokens before
			// and after the separator.
			key := rebuild(tokens[:i]) + keyLeft
			value := valueRight + rebuild(tokens[i+1:])
			return item{
				Key: key,
				Val: value,
				Sep: chosenSep,
				Arg: arg,
			}, nil
		}
	}
	return item{}, fmt.Errorf("%s is not a valid value", arg)
}

// rebuild joins the values of the tokens into a single string,
// preserving the original order.
func rebuild(tokens []token) string {
	var sb strings.Builder
	for _, t := range tokens {
		sb.WriteString(t.value)
	}
	return sb.String()
}

// tokenize tokenize the raw arg string. There are only two [token] types,
// strings and escaped characters, usage example:
//
//	tokenize(`foo\=bar\\baz`, []string{"="})
//
// Result:
//
//	[foo = bar\\baz]
func tokenize(arg string, seps []string) []token {
	var tokens []token
	var current strings.Builder
	for i := 0; i < len(arg); i++ {
		if arg[i] == '\\' {
			if i+1 < len(arg) {
				nextChar := string(arg[i+1])
				if slices.Contains(seps, nextChar) {

					// Save what we have accumulated so far.
					if current.Len() > 0 {
						tokens = append(tokens, token{value: current.String()})
						current.Reset()
					}

					// Save the escaped character as a special token.
					tokens = append(tokens, token{value: nextChar, escaped: true})
					i++ // Skip the next character as it's part of the escape sequence.
					continue
				}
				current.WriteByte('\\')
				current.WriteByte(arg[i+1])
				i++
			} else {
				current.WriteByte('\\')
			}
		} else {
			current.WriteByte(arg[i])
		}
	}
	if current.Len() > 0 {
		tokens = append(tokens, token{value: current.String()})
	}
	return tokens
}
