package processlog

import (
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
)

var (
	errInvalidRecord = errors.New("invalid process log record")
	correlationID    atomic.Uint64
	eventRules       = map[Operation]eventRule{
		ProcessStart:           basicRule(Process, Started),
		ProcessStop:            outcomeRule(Process, map[Outcome]fieldRule{Started: {}, Completed: {}, Failed: {errorCode: true}}),
		ConfigValidate:         outcomeRule(Config, map[Outcome]fieldRule{Succeeded: {}, Failed: {errorCode: true}}),
		StorageMigrate:         outcomeRule(Storage, map[Outcome]fieldRule{Succeeded: {}, Failed: {errorCode: true}}),
		DeclarationPreview:     outcomeRule(Declaration, map[Outcome]fieldRule{Succeeded: {}, Failed: {errorCode: true}}),
		DeclarationApply:       outcomeRule(Declaration, map[Outcome]fieldRule{Succeeded: {}, Failed: {errorCode: true}}),
		HTTPAccept:             requestRule(HTTP, Started, false),
		HTTPReject:             requestRule(HTTP, Rejected, true),
		HTTPComplete:           requestRule(HTTP, Completed, false),
		HTTPFail:               requestRule(HTTP, Failed, true),
		AuthReject:             outcomeRule(Auth, map[Outcome]fieldRule{Rejected: {errorCode: true, correlation: true, protocol: true, authCode: true}}),
		KernelCommandOperation: outcomeRule(Kernel, map[Outcome]fieldRule{Succeeded: commandFields(false), Failed: commandFields(true)}),
		StorageError:           outcomeRule(Storage, map[Outcome]fieldRule{Failed: {errorCode: true}}),
		DependencyError:        outcomeRule(Storage, map[Outcome]fieldRule{Failed: {errorCode: true}}),
	}
)

type eventRule struct {
	component     Component
	outcomeFields map[Outcome]fieldRule
}

type fieldRule struct {
	errorCode     bool
	correlation   bool
	commandFamily bool
	protocol      bool
	authCode      bool
}

func Correlation() string {
	return fmt.Sprintf("p-%016x", correlationID.Add(1))
}

func validate(record Record) error {
	if !knownSeverity(record.Severity) {
		return invalid("severity")
	}
	rule, ok := eventRules[record.Operation]
	if !ok {
		return invalid("operation")
	}
	if record.Component != rule.component {
		return invalid("component")
	}
	fields, ok := rule.outcomeFields[record.Outcome]
	if !ok {
		return invalid("outcome")
	}
	return validateFields(record, fields)
}

func validateFields(record Record, fields fieldRule) error {
	if err := validateCode(record.ErrorCode, fields); err != nil {
		return err
	}
	if err := validateCorrelation(record.Correlation, fields.correlation); err != nil {
		return err
	}
	if err := validateFamily(record.CommandFamily, fields.commandFamily); err != nil {
		return err
	}
	if err := validateProtocol(record.Protocol, fields.protocol); err != nil {
		return err
	}
	return nil
}

func validateCode(code Code, fields fieldRule) error {
	if !fields.errorCode && code != "" {
		return invalid("error_code")
	}
	if fields.errorCode && code == "" {
		return invalid("error_code")
	}
	if code != "" && !knownCode(code) {
		return invalid("error_code")
	}
	if fields.authCode && code != AuthRejected {
		return invalid("error_code")
	}
	return nil
}

func validateCorrelation(value string, required bool) error {
	if !required && value != "" {
		return invalid("correlation")
	}
	if required && !generatedCorrelation(value) {
		return invalid("correlation")
	}
	return nil
}

func validateFamily(value CommandFamily, required bool) error {
	if !required && value != "" {
		return invalid("command_family")
	}
	if required && !knownFamily(value) {
		return invalid("command_family")
	}
	return nil
}

func validateProtocol(value Protocol, required bool) error {
	if !required && value != "" {
		return invalid("protocol")
	}
	if required && value != HTTPProtocol {
		return invalid("protocol")
	}
	return nil
}

func invalid(field string) error {
	return fmt.Errorf("%w: %s", errInvalidRecord, field)
}

func generatedCorrelation(value string) bool {
	if !strings.HasPrefix(value, "p-") || len(value) != 18 {
		return false
	}
	for _, char := range value[2:] {
		if !strings.ContainsRune("0123456789abcdef", char) {
			return false
		}
	}
	return true
}

func basicRule(component Component, outcome Outcome) eventRule {
	return outcomeRule(component, map[Outcome]fieldRule{outcome: {}})
}

func requestRule(component Component, outcome Outcome, errorCode bool) eventRule {
	return outcomeRule(component, map[Outcome]fieldRule{
		outcome: {errorCode: errorCode, correlation: true, protocol: true},
	})
}

func commandFields(errorCode bool) fieldRule {
	return fieldRule{
		errorCode:     errorCode,
		correlation:   true,
		commandFamily: true,
		protocol:      true,
	}
}

func outcomeRule(component Component, outcomes map[Outcome]fieldRule) eventRule {
	return eventRule{component: component, outcomeFields: outcomes}
}

func knownSeverity(value Severity) bool {
	return value == Info || value == Warn || value == Error
}

func knownCode(value Code) bool {
	switch value {
	case AuthRejected,
		InvalidInput,
		UnknownVocabulary,
		EmptyQueue,
		InvalidLease,
		ExpiredLease,
		NoRoute,
		Conflict,
		ConfigInvalid,
		StorageUnavailable,
		StorageMigration,
		DeclarationInvalid,
		DependencyUnavailable,
		InternalError:
		return true
	default:
		return false
	}
}

func knownFamily(value CommandFamily) bool {
	switch value {
	case Submit, Claim, Ack, Nack, Extend, Heartbeat:
		return true
	default:
		return false
	}
}
