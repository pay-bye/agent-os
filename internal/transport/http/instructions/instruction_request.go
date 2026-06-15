package instructions

import (
	"net/http"
	"strings"

	"github.com/pay-bye/agent-os/internal/channel"
	"github.com/pay-bye/agent-os/internal/kernel"
	"github.com/pay-bye/agent-os/internal/registry"
	"github.com/pay-bye/agent-os/internal/transport/http/codec"
	"github.com/pay-bye/agent-os/internal/workitem"
)

const maxInstructionIDs = 100

type pauseInstructionRequest struct {
	ID   string `json:"instruction_id"`
	Node string `json:"node_key"`
}

type leaseInstructionRequest struct {
	ID    string `json:"instruction_id"`
	Lease string `json:"lease_id"`
}

type moveItemInstructionRequest struct {
	ID       string `json:"instruction_id"`
	WorkItem string `json:"work_item_id"`
	Source   string `json:"source_channel_key"`
	Target   string `json:"target_channel_key"`
}

type moveEntriesInstructionRequest struct {
	ID      string   `json:"instruction_id"`
	Source  string   `json:"source_channel_key"`
	Target  string   `json:"target_channel_key"`
	Entries []string `json:"entry_ids"`
}

type moveAvailableInstructionRequest struct {
	ID     string `json:"instruction_id"`
	Source string `json:"source_channel_key"`
	Target string `json:"target_channel_key"`
	Limit  int    `json:"limit"`
}

type itemsInstructionRequest struct {
	ID        string   `json:"instruction_id"`
	WorkItems []string `json:"work_item_ids"`
}

func decodePauseInstruction(request *http.Request) (kernel.PauseInstructionInput, error) {
	var body pauseInstructionRequest
	if err := codec.DecodeBody(request, &body); err != nil {
		return kernel.PauseInstructionInput{}, err
	}
	if err := validateInstructionID(body.ID); err != nil {
		return kernel.PauseInstructionInput{}, err
	}
	if codec.Blank(body.Node) {
		return kernel.PauseInstructionInput{}, codec.ErrInvalidInput
	}
	return kernel.PauseInstructionInput{ID: kernel.InstructionID(body.ID), Node: registry.NodeKey(body.Node)}, nil
}

func decodeLeaseInstruction(request *http.Request) (kernel.LeaseInstructionInput, error) {
	var body leaseInstructionRequest
	if err := codec.DecodeBody(request, &body); err != nil {
		return kernel.LeaseInstructionInput{}, err
	}
	if err := validateInstructionID(body.ID); err != nil {
		return kernel.LeaseInstructionInput{}, err
	}
	if codec.Blank(body.Lease) {
		return kernel.LeaseInstructionInput{}, codec.ErrInvalidInput
	}
	return kernel.LeaseInstructionInput{ID: kernel.InstructionID(body.ID), Lease: channel.LeaseID(body.Lease)}, nil
}

func decodeMoveItemInstruction(request *http.Request) (kernel.MoveItemInstructionInput, error) {
	var body moveItemInstructionRequest
	if err := codec.DecodeBody(request, &body); err != nil {
		return kernel.MoveItemInstructionInput{}, err
	}
	if err := validateInstructionID(body.ID); err != nil {
		return kernel.MoveItemInstructionInput{}, err
	}
	if codec.Blank(body.WorkItem) || codec.Blank(body.Source) || codec.Blank(body.Target) {
		return kernel.MoveItemInstructionInput{}, codec.ErrInvalidInput
	}
	return kernel.MoveItemInstructionInput{
		ID:       kernel.InstructionID(body.ID),
		WorkItem: workitem.ID(body.WorkItem),
		Source:   registry.ChannelKey(body.Source),
		Target:   registry.ChannelKey(body.Target),
	}, nil
}

func decodeMoveEntriesInstruction(request *http.Request) (kernel.MoveEntriesInstructionInput, error) {
	var body moveEntriesInstructionRequest
	if err := codec.DecodeBody(request, &body); err != nil {
		return kernel.MoveEntriesInstructionInput{}, err
	}
	if err := validateInstructionID(body.ID); err != nil {
		return kernel.MoveEntriesInstructionInput{}, err
	}
	entries, err := entryIDs(body.Entries)
	if err != nil {
		return kernel.MoveEntriesInstructionInput{}, err
	}
	if codec.Blank(body.Source) || codec.Blank(body.Target) {
		return kernel.MoveEntriesInstructionInput{}, codec.ErrInvalidInput
	}
	return kernel.MoveEntriesInstructionInput{
		ID:      kernel.InstructionID(body.ID),
		Source:  registry.ChannelKey(body.Source),
		Target:  registry.ChannelKey(body.Target),
		Entries: entries,
	}, nil
}

func decodeMoveAvailableInstruction(request *http.Request) (kernel.MoveAvailableInstructionInput, error) {
	var body moveAvailableInstructionRequest
	if err := codec.DecodeBody(request, &body); err != nil {
		return kernel.MoveAvailableInstructionInput{}, err
	}
	if err := validateInstructionID(body.ID); err != nil {
		return kernel.MoveAvailableInstructionInput{}, err
	}
	if codec.Blank(body.Source) || codec.Blank(body.Target) || body.Limit <= 0 || body.Limit > maxInstructionIDs {
		return kernel.MoveAvailableInstructionInput{}, codec.ErrInvalidInput
	}
	return kernel.MoveAvailableInstructionInput{
		ID:     kernel.InstructionID(body.ID),
		Source: registry.ChannelKey(body.Source),
		Target: registry.ChannelKey(body.Target),
		Limit:  body.Limit,
	}, nil
}

func decodeItemsInstruction(request *http.Request) (kernel.ItemsInstructionInput, error) {
	var body itemsInstructionRequest
	if err := codec.DecodeBody(request, &body); err != nil {
		return kernel.ItemsInstructionInput{}, err
	}
	if err := validateInstructionID(body.ID); err != nil {
		return kernel.ItemsInstructionInput{}, err
	}
	items, err := workItemIDs(body.WorkItems)
	if err != nil {
		return kernel.ItemsInstructionInput{}, err
	}
	return kernel.ItemsInstructionInput{ID: kernel.InstructionID(body.ID), WorkItems: items}, nil
}

func validateInstructionID(id string) error {
	if codec.Blank(id) || len(id) > 128 || strings.ContainsAny(id, " \t\r\n") {
		return codec.ErrInvalidInput
	}
	return nil
}

func entryIDs(values []string) ([]channel.EntryID, error) {
	if err := validateIDs(values); err != nil {
		return nil, err
	}
	ids := make([]channel.EntryID, 0, len(values))
	for _, value := range values {
		ids = append(ids, channel.EntryID(value))
	}
	return ids, nil
}

func workItemIDs(values []string) ([]workitem.ID, error) {
	if err := validateIDs(values); err != nil {
		return nil, err
	}
	ids := make([]workitem.ID, 0, len(values))
	for _, value := range values {
		ids = append(ids, workitem.ID(value))
	}
	return ids, nil
}

func validateIDs(values []string) error {
	if len(values) == 0 || len(values) > maxInstructionIDs {
		return codec.ErrInvalidInput
	}
	seen := map[string]bool{}
	for _, value := range values {
		if codec.Blank(value) || seen[value] {
			return codec.ErrInvalidInput
		}
		seen[value] = true
	}
	return nil
}
