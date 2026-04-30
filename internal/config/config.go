package config

import (
	"fmt"
	"os"
	"path/filepath"
)

var requiredSiblings = []string{
	"uws",
	"udon",
	"symphony",
	"grand",
	"golet",
	"hcllight",
	"horizon",
	"molecule",
	"arazzo",
	"openapisearch",
}

func RequiredSiblings() []string {
	out := make([]string, len(requiredSiblings))
	copy(out, requiredSiblings)
	return out
}

// CheckSiblings verifies that Ramen's required sibling repositories exist.
func CheckSiblings(root string) error {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("resolve root: %w", err)
	}

	parent := filepath.Dir(absRoot)
	for _, name := range requiredSiblings {
		path := filepath.Join(parent, name)
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("required sibling %q not found at %s: %w", name, path, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("required sibling %q at %s is not a directory", name, path)
		}
	}
	return nil
}
