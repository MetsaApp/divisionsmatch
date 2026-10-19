package divisionsmatch_test

import (
	"math"
	"reflect"
	"testing"
)

// The module keeps to the standard library in its tests too; these are
// the few assertions the tests need.

func equal(t *testing.T, want, got any, msg ...any) {
	t.Helper()
	if !reflect.DeepEqual(want, got) {
		t.Errorf("%v\nwant: %#v\n got: %#v", msg, want, got)
	}
}

func near(t *testing.T, want, got float64, msg ...any) {
	t.Helper()
	if math.Abs(want-got) > 0.001 {
		t.Errorf("%v: want %v, got %v", msg, want, got)
	}
}

func must(t *testing.T, ok bool, msg ...any) {
	t.Helper()
	if !ok {
		t.Fatal(msg...)
	}
}
