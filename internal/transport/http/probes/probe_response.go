package probes

import "context"

const (
	Live     Result = "live"
	Ready    Result = "ready"
	NotReady Result = "not_ready"
)

type Result string

type Readiness struct {
	Startup     Result
	Storage     Result
	Migrations  Result
	Verifier    Result
	Declaration Result
	Handler     Result
}

type ReadinessFunc func(context.Context) Readiness

type healthResponse struct {
	Result Result `json:"result"`
}

type readyzResponse struct {
	Result Result         `json:"result"`
	Checks checksResponse `json:"checks"`
}

type checksResponse struct {
	Startup     Result `json:"startup"`
	Storage     Result `json:"storage"`
	Migrations  Result `json:"migrations"`
	Verifier    Result `json:"verifier"`
	Declaration Result `json:"declaration"`
	Handler     Result `json:"handler"`
}

func AllReady() Readiness {
	return Readiness{
		Startup:     Ready,
		Storage:     Ready,
		Migrations:  Ready,
		Verifier:    Ready,
		Declaration: Ready,
		Handler:     Ready,
	}
}

func AllNotReady(context.Context) Readiness {
	return Readiness{
		Startup:     NotReady,
		Storage:     NotReady,
		Migrations:  NotReady,
		Verifier:    NotReady,
		Declaration: NotReady,
		Handler:     NotReady,
	}
}

func healthBody() healthResponse {
	return healthResponse{Result: Live}
}

func readyzBody(readiness Readiness) readyzResponse {
	return readyzResponse{
		Result: readinessResult(readiness),
		Checks: checksResponse(readiness),
	}
}

func readinessResult(readiness Readiness) Result {
	for _, value := range readinessValues(readiness) {
		if value != Ready {
			return NotReady
		}
	}
	return Ready
}

func readinessValues(readiness Readiness) []Result {
	return []Result{
		readiness.Startup,
		readiness.Storage,
		readiness.Migrations,
		readiness.Verifier,
		readiness.Declaration,
		readiness.Handler,
	}
}
