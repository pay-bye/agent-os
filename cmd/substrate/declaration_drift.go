package main

import "github.com/pay-bye/agent-os/internal/declaration"

func hasDeclarationDrift(delta declaration.Delta) bool {
	return len(delta.Additions) > 0 ||
		len(delta.Removals) > 0 ||
		len(delta.Clearances) > 0 ||
		len(delta.Conflicts) > 0
}
