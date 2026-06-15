package invocation_test

import (
	"testing"

	"github.com/pay-bye/agent-os/tests/conformance/schemadoc"
)

func readSchema(t *testing.T, name string) schemadoc.Document {
	t.Helper()

	return schemadoc.Read(t, contractRoot(t), schemaPath(name))
}

func contractRoot(t *testing.T) string {
	t.Helper()

	return schemadoc.FindRoot(t, "contracts", "invocation")
}

func exists(path string) bool {
	return schemadoc.Exists(path)
}
