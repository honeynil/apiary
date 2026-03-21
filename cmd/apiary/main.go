// Command apiary reads annotated Go source files and generates an OpenAPI 3.1
// YAML document.
//
// Usage:
//
//	apiary [flags] [patterns...]
//
// Patterns follow the same ./... convention as the go tool. If no pattern is
// given, "./..." is assumed.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/honeynil/apiary/internal/openapi"
	"github.com/honeynil/apiary/internal/parser"
	"gopkg.in/yaml.v3"
)

func main() {
	out := flag.String("out", "openapi.yaml", "output file path (use - for stdout)")
	title := flag.String("title", "API", "API title")
	version := flag.String("version", "0.0.1", "API version")
	security := flag.String("security", "", "comma-separated global security schemes: bearer, basic, apikey")
	flag.Parse()

	patterns := flag.Args()
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}

	dirs, err := resolvePatterns(patterns)
	if err != nil {
		log.Fatalf("apiary: %v", err)
	}

	p := parser.New()
	for _, dir := range dirs {
		if err := p.ParseDir(dir); err != nil {
			log.Printf("apiary: warning: skipping %s: %v", dir, err)
		}
	}

	ops := p.Operations()
	if len(ops) == 0 {
		fmt.Fprintln(os.Stderr, "apiary: no operations found")
		os.Exit(1)
	}

	builder := openapi.NewBuilder(*title, *version)
	if *security != "" {
		for _, s := range strings.Split(*security, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				builder.WithSecurity(s)
			}
		}
	}
	spec, err := builder.Build(ops, p.Types())
	if err != nil {
		log.Fatalf("apiary: %v", err)
	}

	data, err := yaml.Marshal(spec)
	if err != nil {
		log.Fatalf("apiary: marshal yaml: %v", err)
	}

	if *out == "-" {
		fmt.Print(string(data))
		return
	}

	if dir := filepath.Dir(*out); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			log.Fatalf("apiary: create output dir: %v", err)
		}
	}
	if err := os.WriteFile(*out, data, 0o644); err != nil {
		log.Fatalf("apiary: write file: %v", err)
	}
	fmt.Printf("apiary: wrote %s (%d operation(s))\n", *out, len(ops))
}

// resolvePatterns converts go-tool-style patterns (./..., ./pkg/..., ./pkg)
// into a deduplicated list of directories to scan.
func resolvePatterns(patterns []string) ([]string, error) {
	var dirs []string
	seen := make(map[string]bool)

	add := func(dir string) {
		// Normalise path so that "." and "" both map to ".".
		if dir == "" {
			dir = "."
		}
		if !seen[dir] {
			seen[dir] = true
			dirs = append(dirs, dir)
		}
	}

	for _, pattern := range patterns {
		// Detect recursive pattern (ending in /... or equal to ...)
		recursive := strings.HasSuffix(pattern, "/...") ||
			pattern == "..." ||
			pattern == "./..."

		var baseDir string
		if recursive {
			baseDir = strings.TrimSuffix(pattern, "/...")
			baseDir = strings.TrimSuffix(baseDir, "...")
			baseDir = strings.TrimPrefix(baseDir, "./")
			if baseDir == "" {
				baseDir = "."
			}
		} else {
			baseDir = strings.TrimPrefix(pattern, "./")
			if baseDir == "" {
				baseDir = "."
			}
		}

		if recursive {
			err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return nil // skip unreadable entries
				}
				if info.IsDir() {
					// Skip hidden directories and the vendor directory.
					base := filepath.Base(path)
					if base != "." && (strings.HasPrefix(base, ".") || base == "vendor") {
						return filepath.SkipDir
					}
					add(path)
				}
				return nil
			})
			if err != nil {
				return nil, fmt.Errorf("walk %s: %w", baseDir, err)
			}
		} else {
			add(baseDir)
		}
	}

	return dirs, nil
}
