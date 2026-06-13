package repository

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var unsafeRepositorySQLPatterns = []struct {
	name    string
	pattern *regexp.Regexp
}{
	{name: "raw where string parameter", pattern: regexp.MustCompile(`\bwhere\s+string\b`)},
	{name: "where fragment concatenation", pattern: regexp.MustCompile(`(?:\+\s*where\b|\bwhere\s*\+)`)},
	{name: "fmt.Sprintf SQL assembly", pattern: regexp.MustCompile(`\bfmt\.Sprintf\s*\(`)},
}

func TestRepositorySQLDoesNotAcceptRawFragments(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		path := filepath.Join(".", name)
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		for _, rule := range unsafeRepositorySQLPatterns {
			if rule.pattern.Match(content) {
				t.Fatalf("%s matches unsafe repository SQL pattern %q; use fixed query branches and bind values as parameters", path, rule.name)
			}
		}
	}
}
