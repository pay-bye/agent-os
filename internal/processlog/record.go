package processlog

type Record struct {
	Timestamp     string        `json:"timestamp,omitempty"`
	Severity      Severity      `json:"severity"`
	Component     Component     `json:"component"`
	Operation     Operation     `json:"operation"`
	Outcome       Outcome       `json:"outcome"`
	ErrorCode     Code          `json:"error_code,omitempty"`
	Correlation   string        `json:"correlation,omitempty"`
	CommandFamily CommandFamily `json:"command_family,omitempty"`
	Protocol      Protocol      `json:"protocol,omitempty"`
}

func ProcessStarted() Record {
	return record(Info, Process, ProcessStart, Started)
}

func ProcessStopped(outcome Outcome, code Code) Record {
	return withCode(record(severity(outcome), Process, ProcessStop, outcome), code)
}

func ConfigValidated(outcome Outcome, code Code) Record {
	return withCode(record(severity(outcome), Config, ConfigValidate, outcome), code)
}

func StorageMigrated(outcome Outcome, code Code) Record {
	return withCode(record(severity(outcome), Storage, StorageMigrate, outcome), code)
}

func DeclarationPreviewed(outcome Outcome, code Code) Record {
	return withCode(record(severity(outcome), Declaration, DeclarationPreview, outcome), code)
}

func DeclarationApplied(outcome Outcome, code Code) Record {
	return withCode(record(severity(outcome), Declaration, DeclarationApply, outcome), code)
}

func HTTPAccepted(correlation string) Record {
	return requestRecord(Info, HTTPAccept, Started, correlation, "")
}

func HTTPRejected(correlation string, code Code) Record {
	return requestRecord(Warn, HTTPReject, Rejected, correlation, code)
}

func HTTPFailed(correlation string, code Code) Record {
	return requestRecord(Error, HTTPFail, Failed, correlation, code)
}

func HTTPCompleted(correlation string) Record {
	return requestRecord(Info, HTTPComplete, Completed, correlation, "")
}

func AuthRejectedRecord(correlation string) Record {
	return Record{
		Severity:    Warn,
		Component:   Auth,
		Operation:   AuthReject,
		Outcome:     Rejected,
		ErrorCode:   AuthRejected,
		Correlation: correlation,
		Protocol:    HTTPProtocol,
	}
}

func KernelCommand(family CommandFamily, outcome Outcome, code Code) Record {
	return KernelCommandWithCorrelation(Correlation(), family, outcome, code)
}

func KernelCommandWithCorrelation(
	correlation string,
	family CommandFamily,
	outcome Outcome,
	code Code,
) Record {
	return Record{
		Severity:      severity(outcome),
		Component:     Kernel,
		Operation:     KernelCommandOperation,
		Outcome:       outcome,
		ErrorCode:     safeCode(code),
		Correlation:   correlation,
		CommandFamily: family,
		Protocol:      HTTPProtocol,
	}
}

func StorageFailure(code Code) Record {
	return withCode(record(Error, Storage, StorageError, Failed), code)
}

func DependencyFailure(code Code) Record {
	return withCode(record(Error, Storage, DependencyError, Failed), code)
}

func requestRecord(severity Severity, operation Operation, outcome Outcome, correlation string, code Code) Record {
	return Record{
		Severity:    severity,
		Component:   HTTP,
		Operation:   operation,
		Outcome:     outcome,
		ErrorCode:   safeCode(code),
		Correlation: correlation,
		Protocol:    HTTPProtocol,
	}
}

func record(severity Severity, component Component, operation Operation, outcome Outcome) Record {
	return Record{
		Severity:  severity,
		Component: component,
		Operation: operation,
		Outcome:   outcome,
	}
}

func withCode(record Record, code Code) Record {
	record.ErrorCode = safeCode(code)
	return record
}

func severity(outcome Outcome) Severity {
	switch outcome {
	case Rejected:
		return Warn
	case Failed:
		return Error
	default:
		return Info
	}
}

func safeCode(code Code) Code {
	if code == "" {
		return ""
	}
	if !knownCode(code) {
		return InternalError
	}
	return code
}
