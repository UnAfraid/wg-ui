package subscription

import (
	"path"
	"strings"
)

func joinPath(chunks ...string) string {
	return strings.ReplaceAll(strings.ToLower(path.Join(chunks...)), "/", ".")
}
