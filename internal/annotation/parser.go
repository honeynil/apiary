// Package annotation parses apiary marker comments from Go source files.
package annotation

import (
	"strconv"
	"strings"
)

// Operation holds the parsed metadata for a single API operation.
type Operation struct {
	Method      string
	Path        string
	Summary     string
	Description string
	Tags        []string
	Errors      []int
	// Security lists the scheme names that protect this operation.
	// A single element "none" means explicitly no security (overrides global).
	// Nil means "inherit global security".
	Security []string
	// Request and Response are explicit type names from annotations.
	// Used when the handler signature does not carry type information (e.g. gin).
	// Supports plain names ("UserDTO") and slice syntax ("[]UserDTO").
	Request  string
	Response string
}

// Parse parses comment lines (without the "//" prefix and leading space) into
// an Operation. The slice must contain a line starting with "apiary:operation".
// Returns false if no such marker is found.
func Parse(lines []string) (*Operation, bool) {
	op := &Operation{}
	found := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "apiary:operation ") {
			rest := strings.TrimPrefix(line, "apiary:operation ")
			parts := strings.Fields(rest)
			if len(parts) < 2 {
				continue
			}
			op.Method = strings.ToUpper(parts[0])
			op.Path = parts[1]
			found = true
			continue
		}

		idx := strings.Index(line, ":")
		if idx <= 0 {
			continue
		}

		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])

		switch key {
		case "summary":
			op.Summary = value
		case "description":
			op.Description = value
		case "tags":
			for _, tag := range strings.Split(value, ",") {
				tag = strings.TrimSpace(tag)
				if tag != "" {
					op.Tags = append(op.Tags, tag)
				}
			}
		case "errors":
			for _, code := range strings.Split(value, ",") {
				code = strings.TrimSpace(code)
				if code == "" {
					continue
				}
				n, err := strconv.Atoi(code)
				if err == nil {
					op.Errors = append(op.Errors, n)
				}
			}
		case "security":
			for _, s := range strings.Split(value, ",") {
				s = strings.TrimSpace(s)
				if s != "" {
					op.Security = append(op.Security, s)
				}
			}
		case "request":
			op.Request = value
		case "response":
			op.Response = value
		}
	}

	if !found {
		return nil, false
	}
	return op, true
}
