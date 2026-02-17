package backend

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
)

// ParsedURL represents a parsed backend URL
type ParsedURL struct {
	Type    string            // Backend type (scheme without "://")
	Host    string            // Host for remote backends (e.g., ssh)
	Port    string            // Port for remote backends
	User    string            // Username for remote backends
	Path    string            // Path component
	Options map[string]string // Query parameters as options
	RawURL  string            // Original URL string
}

// ParseURL parses a backend URL and extracts its components
func ParseURL(rawURL string) (*ParsedURL, error) {
	if len(strings.TrimSpace(rawURL)) == 0 {
		return nil, ErrInvalidBackendURL
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidBackendURL, err)
	}

	if parsed.Scheme == "" {
		return nil, fmt.Errorf("%w: missing scheme", ErrInvalidBackendURL)
	}

	backendType := strings.ToLower(parsed.Scheme)

	result := &ParsedURL{
		Type:    backendType,
		Host:    parsed.Hostname(),
		Port:    parsed.Port(),
		Path:    parsed.Path,
		Options: make(map[string]string),
		RawURL:  rawURL,
	}

	if parsed.User != nil {
		result.User = parsed.User.Username()
	}

	for key, values := range parsed.Query() {
		if len(values) > 0 {
			result.Options[key] = values[0]
		}
	}

	return result, nil
}

// BuildURL constructs a backend URL from components
func BuildURL(backendType string, host string, port string, user string, path string, options map[string]string) string {
	var sb strings.Builder
	sb.WriteString(backendType)
	sb.WriteString("://")

	if user != "" {
		sb.WriteString(user)
		sb.WriteString("@")
	}

	if host != "" {
		sb.WriteString(host)
		if port != "" {
			sb.WriteString(":")
			sb.WriteString(port)
		}
	}

	if path != "" {
		if !strings.HasPrefix(path, "/") {
			sb.WriteString("/")
		}
		sb.WriteString(path)
	}

	if len(options) > 0 {
		sb.WriteString("?")
		// Sort keys for deterministic output
		keys := make([]string, 0, len(options))
		for k := range options {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		first := true
		for _, k := range keys {
			if !first {
				sb.WriteString("&")
			}
			sb.WriteString(url.QueryEscape(k))
			sb.WriteString("=")
			sb.WriteString(url.QueryEscape(options[k]))
			first = false
		}
	}

	return sb.String()
}
