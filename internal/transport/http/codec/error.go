package codec

import (
	"errors"
	"github.com/pay-bye/agent-os/internal/channel"
	"github.com/pay-bye/agent-os/internal/kernel"
	"github.com/pay-bye/agent-os/internal/processlog"
	"github.com/pay-bye/agent-os/internal/registry"
	"github.com/pay-bye/agent-os/internal/workitem"
)

var (
	ErrInvalidInput = errors.New("invalid input")
	ErrUnavailable  = errors.New("unavailable")
)

type errorClassification struct {
	status   int
	response string
	code     processlog.Code
}

func Classify(err error) (int, string) {
	classification := classifyError(err)
	return classification.status, classification.response
}

func DiagnosticCode(err error) processlog.Code {
	return classifyError(err).code
}

func classifyError(err error) errorClassification {
	switch {
	case invalidInput(err):
		return errorClassification{status: 400, response: "invalid_input", code: processlog.InvalidInput}
	case registry.IsNotFound(err):
		return errorClassification{status: 404, response: "unknown_vocabulary", code: processlog.UnknownVocabulary}
	case errors.Is(err, channel.ErrEmpty):
		return errorClassification{status: 404, response: "empty_queue", code: processlog.EmptyQueue}
	case errors.Is(err, channel.ErrInvalidLease):
		return errorClassification{status: 404, response: "invalid_lease", code: processlog.InvalidLease}
	case errors.Is(err, channel.ErrExpiredLease):
		return errorClassification{status: 404, response: "expired_lease", code: processlog.ExpiredLease}
	case errors.Is(err, registry.ErrNoRoute):
		return errorClassification{status: 404, response: "no_route", code: processlog.NoRoute}
	default:
		return errorClassification{status: 409, response: "conflict", code: processlog.InternalError}
	}
}

func invalidInput(err error) bool {
	return errors.Is(err, ErrInvalidInput) ||
		errors.Is(err, kernel.ErrInvalidLeaseDuration) ||
		errors.Is(err, workitem.ErrEmptyID) ||
		errors.Is(err, workitem.ErrEmptyKind) ||
		errors.Is(err, workitem.ErrEmptyPayload) ||
		errors.Is(err, workitem.ErrMalformedPayload) ||
		errors.Is(err, workitem.ErrEmptyNeedKind) ||
		errors.Is(err, workitem.ErrMalformedNeedPayload) ||
		errors.Is(err, channel.ErrEmptyLeaseID) ||
		errors.Is(err, channel.ErrMissingExpiresAt) ||
		errors.Is(err, channel.ErrInvalidLeaseWindow) ||
		errors.Is(err, kernel.ErrEmptyInstructionID) ||
		errors.Is(err, kernel.ErrInstructionSelectionEmpty) ||
		errors.Is(err, kernel.ErrInstructionSelectionLimit) ||
		errors.Is(err, kernel.ErrInstructionDuplicateID) ||
		errors.Is(err, kernel.ErrInstructionLimit)
}
