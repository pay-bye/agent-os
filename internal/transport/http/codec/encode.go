package codec

import (
	"bytes"
	"encoding/json"
	nethttp "net/http"
	"time"
)

type errorResponse struct {
	Error string `json:"error"`
}

type Response struct {
	code        int
	body        any
	contentType string
}

func (r Response) StatusCode() int {
	return r.code
}

func WriteOK(response nethttp.ResponseWriter, body any) {
	write(response, 200, body)
}

func WriteError(response nethttp.ResponseWriter, err error) {
	code, token := Classify(err)
	write(response, code, errorResponse{Error: token})
}

func WriteBody(response nethttp.ResponseWriter, result Response) {
	if result.contentType == "" {
		write(response, result.code, result.body)
		return
	}
	response.Header().Set("Content-Type", result.contentType)
	response.WriteHeader(result.code)
	if body, ok := result.body.(string); ok {
		_, _ = response.Write([]byte(body))
	}
}

func ResponsePayload(value []byte) (json.RawMessage, error) {
	trimmed := bytes.TrimSpace(value)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return nil, ErrInvalidInput
	}
	var compacted bytes.Buffer
	if err := json.Compact(&compacted, trimmed); err != nil {
		return nil, err
	}
	return json.RawMessage(compacted.Bytes()), nil
}

func FormatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339)
}

func OK(body any) Response {
	return WithCode(nethttp.StatusOK, body)
}

func WithCode(code int, body any) Response {
	return Response{code: code, body: body}
}

func TextOK(body string) Response {
	return Response{
		code:        nethttp.StatusOK,
		body:        body,
		contentType: "text/plain; version=0.0.4; charset=utf-8",
	}
}

func write(response nethttp.ResponseWriter, code int, body any) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(code)
	_ = json.NewEncoder(response).Encode(body)
}
