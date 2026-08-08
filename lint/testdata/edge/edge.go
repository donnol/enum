// Package edge is a testdata fixture covering edge cases:
//   - IncDecStmt (X++)
//   - compound assignment (X += ...)
//   - multiple LHS (X.A, X.B = ...)
//   - read access (not flagged)
//   - method calls (not flagged)
//   - lowercase fields (not tracked)
package edge

import "github.com/donnol/enum"

type Mode int

var Modes = enum.InitFor[Mode, struct {
	enum.Struct[Mode]
	Read  Mode `enum:"1,读"`
	Write Mode `enum:"2,写"`
	Count Mode `enum:"3,计数"`
	lower Mode `enum:"4,小写"` // lowercase — should not be tracked
}]()

func incDecWrite() {
	Modes.Count++
}

func compoundAssign() {
	Modes.Count += 1
}

func multipleLHS() {
	Modes.Read, Modes.Write = 2, 1
}

func legitimateReads() {
	_ = Modes.Read   // read — not flagged
	x := Modes.Write // read — not flagged
	_ = x
	_ = Modes.All()        // method call — not flagged
	_ = Modes.Len()        // method call — not flagged
	if Modes.Contains(1) { // method call — not flagged
	}
}

func nonEnumReassign() {
	// A local variable named `Modes` — `:=` creates a new local.
	// Should NOT be flagged.
	Modes := 5
	_ = Modes
}
