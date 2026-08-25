// Package sourcecatalog owns the canonical authoring source-directory list.
package sourcecatalog

import "strings"

var apiDirectories = []string{
	"openapi", "google-discovery", "discovery", "aws-smithy", "asyncapi",
	"graphql", "openrpc", "grpc-protobuf", "odata",
}

var browserDirectories = []string{
	"browser-profiles", "browser-authentication", "browser-registration", "capability-bundles",
}

func API() []string     { return append([]string(nil), apiDirectories...) }
func Browser() []string { return append([]string(nil), browserDirectories...) }
func All() []string     { return append(API(), browserDirectories...) }

// IsAPIPath reports whether the first slash-separated component names a
// canonical API-source directory.
func IsAPIPath(value string) bool {
	value = strings.Trim(strings.TrimSpace(strings.ReplaceAll(value, `\`, "/")), "/")
	first, _, _ := strings.Cut(value, "/")
	for _, directory := range apiDirectories {
		if first == directory {
			return true
		}
	}
	return false
}

// ContainsAPIPath reports whether any component names a canonical API-source
// directory. It is used when extracting source hints that may be absolute.
func ContainsAPIPath(value string) bool {
	value = strings.ReplaceAll(strings.TrimSpace(value), `\`, "/")
	for _, component := range strings.Split(value, "/") {
		for _, directory := range apiDirectories {
			if component == directory {
				return true
			}
		}
	}
	return false
}
