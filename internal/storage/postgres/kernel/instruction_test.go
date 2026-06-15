package kernel

import (
	"reflect"
	"testing"
)

func TestTextArrayScansDriverValues(t *testing.T) {
	var values textArray

	if err := values.Scan("{x80,x81}"); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual([]string(values), []string{"x80", "x81"}) {
		t.Fatalf("values = %v, want x80/x81", values)
	}
}
