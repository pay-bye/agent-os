package compatibility

type Compatibility struct {
	ContractVersion string               `json:"contract_version"`
	SchemaSetDigest string               `json:"schema_set_digest"`
	Features        []string             `json:"features"`
	Routes          []CompatibilityRoute `json:"routes"`
}

type CompatibilityRoute struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

func CompatibilityContract() Compatibility {
	return Compatibility{
		ContractVersion: "v1",
		SchemaSetDigest: "sha256:e08a34f1c973fc0f52336bb66686ceb9c70a0073080aa7fd53ce6a1e22be0325",
		Features: []string{
			"lease_claim",
			"lease_extend",
			"lease_ack",
			"lease_nack",
			"lease_capability",
			"declared_needs",
			"failure_payload",
		},
		Routes: []CompatibilityRoute{
			{Method: "POST", Path: "/submit"},
			{Method: "POST", Path: "/claim"},
			{Method: "POST", Path: "/ack"},
			{Method: "POST", Path: "/nack"},
			{Method: "POST", Path: "/extend"},
			{Method: "POST", Path: "/heartbeat"},
			{Method: "POST", Path: "/operations/instructions/pause"},
			{Method: "POST", Path: "/operations/instructions/release-expired-lease"},
			{Method: "POST", Path: "/operations/instructions/force-release-lease"},
			{Method: "POST", Path: "/operations/instructions/move-item"},
			{Method: "POST", Path: "/operations/instructions/move-entries"},
			{Method: "POST", Path: "/operations/instructions/move-available"},
			{Method: "POST", Path: "/operations/instructions/drop"},
			{Method: "POST", Path: "/operations/instructions/route-outstanding"},
			{Method: "GET", Path: "/health"},
			{Method: "GET", Path: "/readyz"},
			{Method: "GET", Path: "/metrics"},
			{Method: "GET", Path: "/operations"},
			{Method: "GET", Path: "/operations/channels"},
			{Method: "GET", Path: "/operations/channels/{channel_key}/items"},
			{Method: "GET", Path: "/operations/items/{work_item_id}"},
			{Method: "GET", Path: "/operations/items/{work_item_id}/journal"},
			{Method: "GET", Path: "/operations/nodes"},
			{Method: "GET", Path: "/compatibility"},
		},
	}
}

func body() Compatibility {
	return CompatibilityContract()
}
