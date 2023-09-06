package directive

import (
	"github.com/UnAfraid/wg-ui/pkg/api/internal/resolver"
)

func NewDirectiveRoot() resolver.DirectiveRoot {
	return resolver.DirectiveRoot{
		Authenticated: authenticated,
	}
}
