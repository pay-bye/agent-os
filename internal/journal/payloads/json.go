package payloads

import "encoding/json"

func marshal(value map[string]any) ([]byte, error) {
	return json.Marshal(value)
}

func raw(value []byte) json.RawMessage {
	return json.RawMessage(append([]byte(nil), value...))
}
