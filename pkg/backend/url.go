package backend

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

const redactedURLPassword = "***"

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

// RedactURLPassword replaces URL password with a redaction marker.
func RedactURLPassword(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.User == nil {
		return rawURL
	}

	password, hasPassword := parsed.User.Password()
	if !hasPassword || password == "" {
		return rawURL
	}

	parsed.User = url.UserPassword(parsed.User.Username(), redactedURLPassword)
	return parsed.String()
}

// ReplaceRedactedURLPassword replaces redacted password marker with the existing URL password.
func ReplaceRedactedURLPassword(rawURL string, existingURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.User == nil {
		return rawURL, nil
	}

	password, hasPassword := parsed.User.Password()
	if !hasPassword || !isRedactedURLPassword(password) {
		return rawURL, nil
	}

	existingParsed, err := url.Parse(existingURL)
	if err != nil {
		return "", fmt.Errorf("invalid existing backend url: %w", err)
	}
	if existingParsed.User == nil {
		return "", errors.New("redacted password placeholder cannot be used without existing credentials")
	}

	existingPassword, hasExistingPassword := existingParsed.User.Password()
	if !hasExistingPassword || existingPassword == "" {
		return "", errors.New("redacted password placeholder requires an existing password")
	}

	parsed.User = url.UserPassword(parsed.User.Username(), existingPassword)
	return parsed.String(), nil
}

func isRedactedURLPassword(password string) bool {
	current := strings.TrimSpace(password)
	if current == "" {
		return false
	}

	for i := 0; i < 4; i++ {
		if current == redactedURLPassword {
			return true
		}

		decoded, err := url.PathUnescape(current)
		if err != nil || decoded == current {
			break
		}
		current = decoded
	}

	return current == redactedURLPassword
}
