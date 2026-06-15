package registry

import (
	"errors"
)

var (
	ErrEmptyChannelKey         = errors.New("channel key is empty")
	ErrEmptyChannelDescription = errors.New("channel description is empty")
)

type ChannelKey string

func (k ChannelKey) String() string {
	return string(k)
}

type ChannelInput struct {
	Key         ChannelKey
	Node        NodeKey
	Description string
}

type Channel struct {
	key         ChannelKey
	node        NodeKey
	description string
}

func NewChannel(input ChannelInput) (Channel, error) {
	if err := validateChannelInput(input); err != nil {
		return Channel{}, err
	}
	return Channel{key: input.Key, node: input.Node, description: input.Description}, nil
}

func (c Channel) Key() ChannelKey {
	return c.key
}

func (c Channel) Node() NodeKey {
	return c.node
}

func (c Channel) Description() string {
	return c.description
}

func validateChannelInput(input ChannelInput) error {
	if blank(input.Key.String()) {
		return ErrEmptyChannelKey
	}
	if blank(input.Node.String()) {
		return ErrEmptyNodeKey
	}
	if blank(input.Description) {
		return ErrEmptyChannelDescription
	}
	return nil
}
