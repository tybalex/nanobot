package server

import (
	"regexp"
)

var uriTemplateRegex = regexp.MustCompile(`\{([^/]+?)}`)

func uriToRegexp(template string) (*regexp.Regexp, error) {
	// This is naive and stupid and doesn't work right because it doesn't escape the regexp
	reStr := uriTemplateRegex.ReplaceAllString(template, `([^/]+)`)
	reStr = "^" + reStr + "$"

	return regexp.Compile(reStr)
}
