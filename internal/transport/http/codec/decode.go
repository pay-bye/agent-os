package codec

import (
	"bytes"
	"encoding/json"
	"github.com/pay-bye/agent-os/internal/channel"
	"github.com/pay-bye/agent-os/internal/kernel"
	"github.com/pay-bye/agent-os/internal/registry"
	"github.com/pay-bye/agent-os/internal/workitem"
	"io"
	"mime"
	nethttp "net/http"
	"strings"
)

type Object []byte

func (p *Object) UnmarshalJSON(value []byte) error {
	trimmed := bytes.TrimSpace(value)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return ErrInvalidInput
	}
	var compacted bytes.Buffer
	if err := json.Compact(&compacted, trimmed); err != nil {
		return ErrInvalidInput
	}
	*p = compacted.Bytes()
	return nil
}

type NeedRequest struct {
	NeedKind string  `json:"need_kind"`
	Target   *string `json:"target_node"`
	Payload  Object  `json:"payload"`
}

func PayloadBytes(value Object) []byte {
	if len(value) == 0 {
		return nil
	}
	return append([]byte(nil), value...)
}

func PayloadMissing(value Object) bool {
	return len(value) == 0
}

func DecodeBody(request *nethttp.Request, target any) error {
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return ErrInvalidInput
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return ErrInvalidInput
	}
	return nil
}

func DeclaredNeeds(requests []NeedRequest) ([]workitem.DeclaredNeedInput, error) {
	if requests == nil {
		return nil, ErrInvalidInput
	}
	needs := make([]workitem.DeclaredNeedInput, 0, len(requests))
	for _, request := range requests {
		if Blank(request.NeedKind) {
			return nil, ErrInvalidInput
		}
		target, err := targetNode(request.Target)
		if err != nil {
			return nil, err
		}
		needs = append(needs, workitem.DeclaredNeedInput{
			Kind:    registry.NeedKindKey(request.NeedKind),
			Target:  target,
			Payload: PayloadBytes(request.Payload),
		})
	}
	return needs, nil
}

func ResolutionInput(
	lease string,
	rawToken string,
	requests []NeedRequest,
	failurePayload []byte,
) (kernel.ResolutionInput, error) {
	needs, err := DeclaredNeeds(requests)
	if err != nil {
		return kernel.ResolutionInput{}, err
	}
	if Blank(lease) {
		return kernel.ResolutionInput{}, ErrInvalidInput
	}
	token, err := channel.NewToken(rawToken)
	if err != nil {
		return kernel.ResolutionInput{}, ErrInvalidInput
	}
	return kernel.ResolutionInput{
		Lease:          channel.LeaseID(lease),
		Token:          token,
		DeclaredNeeds:  needs,
		FailurePayload: failurePayload,
	}, nil
}

func Blank(value string) bool {
	return strings.TrimSpace(value) == ""
}

func JSONMediaType(header string) bool {
	mediaType, _, err := mime.ParseMediaType(header)
	return err == nil && mediaType == "application/json"
}

func targetNode(value *string) (registry.NodeKey, error) {
	if value == nil {
		return "", nil
	}
	if Blank(*value) {
		return "", ErrInvalidInput
	}
	return registry.NodeKey(*value), nil
}
