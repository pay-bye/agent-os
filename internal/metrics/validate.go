package metrics

func ClosedSeriesCount() int {
	requestCounters := len(requestOperations()) * len(requestResults())
	requestBuckets := requestCounters * len(buckets())
	authCounters := 1
	storageGauges := 3
	journalCounters := len(eventKinds())
	routingCounters := len(outcomes())
	declarationCounters := len(declarationOperations()) * len(operationResults())
	declarationBuckets := declarationCounters * len(buckets())
	migrationCounters := len(operationResults())
	migrationBuckets := migrationCounters * len(buckets())
	processGauge := 1
	buildGauge := 1

	return requestCounters +
		requestBuckets +
		authCounters +
		storageGauges +
		journalCounters +
		routingCounters +
		declarationCounters +
		declarationBuckets +
		migrationCounters +
		migrationBuckets +
		processGauge +
		buildGauge
}

func validRequest(operation Operation, result Result) bool {
	return contains(requestOperations(), operation) && contains(requestResults(), result)
}

func validDeclaration(operation DeclarationOperation, result Result) bool {
	return contains(declarationOperations(), operation) && validOperationResult(result)
}

func validOperationResult(result Result) bool {
	return contains(operationResults(), result)
}

func validEventKind(kind EventKind) bool {
	return contains(eventKinds(), kind)
}

func validOutcome(outcome Outcome) bool {
	return contains(outcomes(), outcome)
}

func contains[T comparable](values []T, value T) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}

func buckets() []bucket {
	return []bucket{
		{label: "0.005", value: 0.005},
		{label: "0.01", value: 0.01},
		{label: "0.025", value: 0.025},
		{label: "0.05", value: 0.05},
		{label: "0.1", value: 0.1},
		{label: "0.25", value: 0.25},
		{label: "1", value: 1},
	}
}
