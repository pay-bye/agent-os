package commands

import (
	"github.com/pay-bye/agent-os/internal/kernel"
	"github.com/pay-bye/agent-os/internal/registry"
	"github.com/pay-bye/agent-os/internal/transport/http/codec"
	"github.com/pay-bye/agent-os/internal/workitem"
	nethttp "net/http"
)

type submitRequest struct {
	WorkItem      string              `json:"work_item_id"`
	ItemKind      string              `json:"item_kind"`
	Payload       codec.Object        `json:"payload"`
	DeclaredNeeds []codec.NeedRequest `json:"declared_needs"`
}

func decodeSubmit(request *nethttp.Request) (kernel.SubmitInput, error) {
	var body submitRequest
	if err := codec.DecodeBody(request, &body); err != nil {
		return kernel.SubmitInput{}, err
	}
	needs, err := codec.DeclaredNeeds(body.DeclaredNeeds)
	if err != nil {
		return kernel.SubmitInput{}, err
	}
	if codec.Blank(body.WorkItem) || codec.Blank(body.ItemKind) || codec.PayloadMissing(body.Payload) {
		return kernel.SubmitInput{}, codec.ErrInvalidInput
	}
	return kernel.SubmitInput{
		ID:            workitem.ID(body.WorkItem),
		Kind:          registry.ItemKindKey(body.ItemKind),
		Payload:       codec.PayloadBytes(body.Payload),
		DeclaredNeeds: needs,
	}, nil
}
