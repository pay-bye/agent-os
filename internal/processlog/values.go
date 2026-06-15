package processlog

const (
	Info  Severity = "info"
	Warn  Severity = "warn"
	Error Severity = "error"
)

const (
	Process     Component = "process"
	Config      Component = "config"
	Storage     Component = "storage"
	Declaration Component = "declaration"
	HTTP        Component = "http"
	Auth        Component = "auth"
	Kernel      Component = "kernel"
)

const (
	ProcessStart           Operation = "process.start"
	ProcessStop            Operation = "process.stop"
	ConfigValidate         Operation = "config.validate"
	StorageMigrate         Operation = "storage.migrate"
	DeclarationPreview     Operation = "declaration.preview"
	DeclarationApply       Operation = "declaration.apply"
	HTTPAccept             Operation = "http.accept"
	HTTPReject             Operation = "http.reject"
	HTTPComplete           Operation = "http.complete"
	HTTPFail               Operation = "http.fail"
	AuthReject             Operation = "auth.reject"
	KernelCommandOperation Operation = "kernel.command"
	StorageError           Operation = "storage.error"
	DependencyError        Operation = "dependency.error"
)

const (
	Started   Outcome = "started"
	Succeeded Outcome = "succeeded"
	Rejected  Outcome = "rejected"
	Failed    Outcome = "failed"
	Completed Outcome = "completed"
)

const (
	Submit      CommandFamily = "submit"
	Claim       CommandFamily = "claim"
	Ack         CommandFamily = "ack"
	Nack        CommandFamily = "nack"
	Extend      CommandFamily = "extend"
	Heartbeat   CommandFamily = "heartbeat"
	Instruction CommandFamily = "instruction"
)

const HTTPProtocol Protocol = "http"

const (
	AuthRejected          Code = "auth.rejected"
	InvalidInput          Code = "invalid.input"
	UnknownVocabulary     Code = "unknown.vocabulary"
	EmptyQueue            Code = "empty.queue"
	InvalidLease          Code = "invalid.lease"
	ExpiredLease          Code = "expired.lease"
	NoRoute               Code = "no.route"
	Conflict              Code = "conflict"
	ConfigInvalid         Code = "config.invalid"
	StorageUnavailable    Code = "storage.unavailable"
	StorageMigration      Code = "storage.migration"
	DeclarationInvalid    Code = "declaration.invalid"
	DependencyUnavailable Code = "dependency.unavailable"
	InternalError         Code = "internal.error"
)

type Severity string
type Component string
type Operation string
type Outcome string
type Code string
type CommandFamily string
type Protocol string
