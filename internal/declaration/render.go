package declaration

import (
	"encoding/json"
)

func Render(delta Delta) ([]byte, error) {
	content, err := json.MarshalIndent(delta, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(content, '\n'), nil
}
