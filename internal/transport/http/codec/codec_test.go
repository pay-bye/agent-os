package codec

import (
	"errors"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pay-bye/agent-os/internal/channel"
	"github.com/pay-bye/agent-os/internal/registry"
)

func TestDecodeBodyRejectsUnknownAndTrailingJSON(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "unknown", body: `{"value":"x1","extra":true}`},
		{name: "trailing", body: `{"value":"x1"} {}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(nethttp.MethodPost, "/x", strings.NewReader(test.body))
			var target struct {
				Value string `json:"value"`
			}

			err := DecodeBody(request, &target)

			if !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("error = %v, want invalid input", err)
			}
		})
	}
}

func TestObjectPreservesCompactPayloadObject(t *testing.T) {
	var value Object

	if err := value.UnmarshalJSON([]byte(`{"value": "x1"}`)); err != nil {
		t.Fatal(err)
	}

	if string(PayloadBytes(value)) != `{"value":"x1"}` {
		t.Fatalf("payload = %s", value)
	}
	if PayloadMissing(value) {
		t.Fatal("payload reported missing")
	}
}

func TestDeclaredNeedsRequiresTargetWhenPresent(t *testing.T) {
	_, err := DeclaredNeeds([]NeedRequest{{
		NeedKind: "x12",
		Target:   stringPointer(""),
		Payload:  Object(`{"value":"x1"}`),
	}})

	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("error = %v, want invalid input", err)
	}
}

func TestResolutionInputBuildsKernelTokenInput(t *testing.T) {
	input, err := ResolutionInput("x16", "x-token", []NeedRequest{{
		NeedKind: "x12",
		Payload:  Object(`{"value":"x1"}`),
	}}, []byte(`{"failure":"x2"}`))
	if err != nil {
		t.Fatal(err)
	}

	if input.Token != channel.Token("x-token") {
		t.Fatalf("token = %q, want x-token", input.Token)
	}
	if len(input.DeclaredNeeds) != 1 || input.DeclaredNeeds[0].Kind != registry.NeedKindKey("x12") {
		t.Fatalf("declared needs = %+v", input.DeclaredNeeds)
	}
	if string(input.FailurePayload) != `{"failure":"x2"}` {
		t.Fatalf("failure payload = %s", input.FailurePayload)
	}
}

func TestWriteBodyPreservesTextResponse(t *testing.T) {
	response := httptest.NewRecorder()

	WriteBody(response, TextOK("sample\n"))

	if response.Code != nethttp.StatusOK {
		t.Fatalf("code = %d, want 200", response.Code)
	}
	if got := response.Header().Get("Content-Type"); got != "text/plain; version=0.0.4; charset=utf-8" {
		t.Fatalf("content type = %q", got)
	}
	if response.Body.String() != "sample\n" {
		t.Fatalf("body = %q", response.Body.String())
	}
}

func TestClassifyKnownErrors(t *testing.T) {
	code, token := Classify(registry.ErrNoRoute)

	if code != nethttp.StatusNotFound || token != "no_route" {
		t.Fatalf("classification = %d %s", code, token)
	}
}

func stringPointer(value string) *string {
	return &value
}
