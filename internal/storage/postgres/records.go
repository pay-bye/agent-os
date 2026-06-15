package postgres

type Catalog struct {
	Schemas           []SchemaRecord
	Items             []ItemRecord
	Needs             []NeedRecord
	Nodes             []NodeRecord
	Routes            []RouteRecord
	RoutingExclusions []RoutingExclusionRecord
}

type SchemaRecord struct {
	Key      string
	Document []byte
}

type ItemRecord struct {
	Key         string
	Schema      string
	Description string
}

type NeedRecord struct {
	Key         string
	Schema      string
	HasSchema   bool
	Description string
}

type NodeRecord struct {
	Key          string
	Description  string
	Accepts      []string
	Channel      string
	ChannelLabel string
}

type RouteRecord struct {
	Need  string
	Node  string
	Order int
}

type RoutingExclusionRecord struct {
	Node string
}
