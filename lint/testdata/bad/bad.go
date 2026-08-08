// Package bad is a testdata fixture: writes to enum vars/fields in
// several ways. The lint tool should flag every write.
package bad

import "github.com/donnol/enum"

type Status string

var Statuses = enum.InitFor[Status, struct {
	enum.Struct[Status]
	Pending Status `enum:"pending,待处理"`
	Active  Status `enum:"active,活跃"`
}]()

func violationFieldWrite() {
	Statuses.Pending = "hacked"
}

func violationVarReassign() {
	Statuses = struct {
		enum.Struct[Status]
		Pending Status `enum:"pending,待处理"`
		Active  Status `enum:"active,活跃"`
	}{}
}

func legitimateShadow() {
	// `:=` creates a new local — not a write to the package-level enum.
	// Should NOT be flagged.
	Statuses := struct {
		enum.Struct[Status]
		Pending Status `enum:"pending,待处理"`
	}{}
	_ = Statuses
}
