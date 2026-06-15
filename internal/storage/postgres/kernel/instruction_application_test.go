package kernel

import (
	"context"
	"testing"

	"github.com/pay-bye/agent-os/internal/kernel/instructions"
)

func TestPersistEmptyInstructionApplicationReturnsResult(t *testing.T) {
	want := instructions.Result{ID: "x70", Result: instructions.Applied}

	got, err := persistInstructionApplication(context.Background(), nil, instructions.Application{Result: want})

	if err != nil {
		t.Fatal(err)
	}
	if got.ID != want.ID || got.Result != want.Result {
		t.Fatalf("result = %+v, want %+v", got, want)
	}
}

func TestPersistEmptyMoveInstructionApplicationReturnsResult(t *testing.T) {
	want := instructions.Result{ID: "x70", Result: instructions.Applied}
	application := instructions.Application{
		Result:  want,
		Effects: instructions.Effects{Moves: []instructions.MoveEffect{{}}},
	}

	got, err := persistInstructionApplication(context.Background(), nil, application)

	if err != nil {
		t.Fatal(err)
	}
	if got.ID != want.ID || got.Result != want.Result {
		t.Fatalf("result = %+v, want %+v", got, want)
	}
}

func TestEmptyInstructionApplicationEffectsRequireNoTransaction(t *testing.T) {
	if err := applyInstructionEffects(context.Background(), nil, instructions.Effects{}); err != nil {
		t.Fatal(err)
	}
}

func TestEmptyMoveInstructionEffectRequiresNoTransaction(t *testing.T) {
	effects := instructions.Effects{Moves: []instructions.MoveEffect{{}}}

	if err := applyInstructionEffects(context.Background(), nil, effects); err != nil {
		t.Fatal(err)
	}
}

func TestEmptyInstructionOutcomesRequireNoTransaction(t *testing.T) {
	if err := appendInstructionOutcomes(context.Background(), nil, nil); err != nil {
		t.Fatal(err)
	}
}

func TestMalformedInstructionOutcomeFailsBeforeTransactionUse(t *testing.T) {
	err := appendInstructionOutcomes(context.Background(), nil, []instructions.Outcome{{}})

	if err == nil {
		t.Fatal("malformed outcome passed")
	}
}

func TestEmptyInstructionRoutesRequireNoTransaction(t *testing.T) {
	if err := applyInstructionRoutes(context.Background(), nil, nil); err != nil {
		t.Fatal(err)
	}
}
