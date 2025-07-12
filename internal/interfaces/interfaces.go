package interfaces

import (
	"github.com/rsteube/carapace"
)

// Completer is the interface for types that can provide their own shell
// completion suggestions.
type Completer interface {
	Complete(ctx carapace.Context) carapace.Action
}
