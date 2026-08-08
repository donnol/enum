package enum

import (
	"testing"
)

// TestInitStruct_NonPointerPanics verifies that passing a non-pointer
// value to initStruct panics at the reflect.ValueOf(val).Elem() call
// before reaching the CanAddr defense. This is the natural consequence
// of the API contract: InitFor always passes &zero, so misuse of
// initStruct with a non-pointer is caught early by the runtime.
func TestInitStruct_NonPointerPanics(t *testing.T) {
	type Example struct {
		Struct[string]
		Alpha string `enum:"1,甲"`
	}

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic when passing non-pointer to initStruct")
		}
		// The panic comes from reflect.ValueOf(val).Elem() on line 41,
		// before the CanAddr check. This is fine — Elem on a non-pointer
		// is the first and clearest failure.
	}()

	var val Example
	initStruct[string](val) // deliberate: pass value, not pointer
}
