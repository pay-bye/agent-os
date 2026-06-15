package registry

import (
	"context"
)

type Reader interface {
	FindSchemaDocument(context.Context, SchemaKey) (SchemaDocument, error)
	FindItemKind(context.Context, ItemKindKey) (ItemKind, error)
	FindNeedKind(context.Context, NeedKindKey) (NeedKind, error)
	FindNode(context.Context, NodeKey) (Node, error)
	FindChannel(context.Context, ChannelKey) (Channel, error)
	FindJournalEventKind(context.Context, JournalEventKindKey) (JournalEventKind, error)
	FindRoutingRules(context.Context, NeedKindKey) ([]RoutingRule, error)
}
