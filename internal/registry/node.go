package registry

import (
	"errors"
)

var (
	ErrEmptyNodeKey         = errors.New("node key is empty")
	ErrEmptyNodeDescription = errors.New("node description is empty")
	ErrEmptyCapabilities    = errors.New("node capabilities are empty")
	ErrEmptyCapabilityKind  = errors.New("node capability kind is empty")
)

type NodeKey string

func (k NodeKey) String() string {
	return string(k)
}

type NodeInput struct {
	Key          NodeKey
	Description  string
	Channel      ChannelKey
	Capabilities []NeedKindKey
}

type Node struct {
	key          NodeKey
	description  string
	channel      ChannelKey
	capabilities []NeedKindKey
}

func NewNode(input NodeInput) (Node, error) {
	if err := validateNodeInput(input); err != nil {
		return Node{}, err
	}
	return Node{
		key:          input.Key,
		description:  input.Description,
		channel:      input.Channel,
		capabilities: copyCapabilities(input.Capabilities),
	}, nil
}

func (n Node) Key() NodeKey {
	return n.key
}

func (n Node) Description() string {
	return n.description
}

func (n Node) Channel() ChannelKey {
	return n.channel
}

func (n Node) Capabilities() []NeedKindKey {
	return copyCapabilities(n.capabilities)
}

func (n Node) HasCapability(need NeedKindKey) bool {
	for _, capability := range n.capabilities {
		if capability == need {
			return true
		}
	}
	return false
}

func validateNodeInput(input NodeInput) error {
	if blank(input.Key.String()) {
		return ErrEmptyNodeKey
	}
	if blank(input.Description) {
		return ErrEmptyNodeDescription
	}
	if blank(input.Channel.String()) {
		return ErrEmptyChannelKey
	}
	return validateCapabilities(input.Capabilities)
}

func validateCapabilities(capabilities []NeedKindKey) error {
	if len(capabilities) == 0 {
		return ErrEmptyCapabilities
	}
	for _, capability := range capabilities {
		if blank(capability.String()) {
			return ErrEmptyCapabilityKind
		}
	}
	return nil
}

func copyCapabilities(capabilities []NeedKindKey) []NeedKindKey {
	return append([]NeedKindKey(nil), capabilities...)
}
