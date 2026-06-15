package main

import (
	"testing"

	"github.com/pay-bye/agent-os/internal/declaration"
)

func TestHasDeclarationDrift(t *testing.T) {
	cases := []struct {
		name  string
		delta declaration.Delta
		want  bool
	}{
		{name: "steady", want: false},
		{
			name:  "addition",
			delta: declaration.Delta{Additions: []declaration.RecordRef{{Kind: "node", Key: "x17"}}},
			want:  true,
		},
		{
			name:  "removal",
			delta: declaration.Delta{Removals: []declaration.RecordRef{{Kind: "node", Key: "x18"}}},
			want:  true,
		},
		{
			name: "clearance",
			delta: declaration.Delta{
				Clearances: []declaration.RecordRef{{Kind: "routing_exclusion", Key: "x17"}},
			},
			want: true,
		},
		{
			name: "conflict",
			delta: declaration.Delta{
				Conflicts: []declaration.RecordConflict{{Kind: "need", Key: "x12", State: "different"}},
			},
			want: true,
		},
	}

	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			got := hasDeclarationDrift(item.delta)
			if got != item.want {
				t.Fatalf("drift = %v, want %v", got, item.want)
			}
		})
	}
}
