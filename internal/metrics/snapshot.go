package metrics

type CounterSnapshot struct {
	Commands CommandCounts
	Routing  RoutingCounts
	Build    Build
}

type CommandCounts struct {
	Succeeded int
	Failed    int
}

type RoutingCounts struct {
	Routed   int
	Unrouted int
}

func commandCounts(requests map[requestKey]int) CommandCounts {
	return CommandCounts{
		Succeeded: commandRequests(requests, Completed),
		Failed:    commandRequests(requests, Failed),
	}
}

func commandRequests(requests map[requestKey]int, result Result) int {
	total := 0
	for _, operation := range commandOperations() {
		total += requests[requestKey{operation: operation, result: result}]
	}
	return total
}

func commandOperations() []Operation {
	return []Operation{Submit, Claim, Ack, Nack, Extend, Heartbeat}
}
