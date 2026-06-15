package instructions

import (
	"testing"

	"github.com/pay-bye/agent-os/internal/workitem"
)

func TestDropRejectsEmptySelection(t *testing.T) {
	_, err := Drop(ItemsInput{ID: "x70"}, fixedClock{now: instant(0)}, &sequenceIDs{})

	requireError(t, err, ErrSelectionEmpty)
}

func TestDropRejectsSelectionLimit(t *testing.T) {
	_, err := Drop(ItemsInput{
		ID:        "x70",
		WorkItems: overBudgetItems(),
	}, fixedClock{now: instant(0)}, &sequenceIDs{})

	requireError(t, err, ErrSelectionLimit)
}

func TestSelectionRejectsEmptyWorkItemIdentity(t *testing.T) {
	_, err := Drop(ItemsInput{
		ID:        "x70",
		WorkItems: []workitem.ID{""},
	}, fixedClock{now: instant(0)}, &sequenceIDs{})

	requireError(t, err, ErrSelectionEmpty)
}

func overBudgetItems() []workitem.ID {
	items := make([]workitem.ID, maxIDs+1)
	for index := range items {
		items[index] = workitem.ID("x08")
	}
	return items
}
