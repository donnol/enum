package shadow

import "github.com/donnol/enum"

type Status string

var Statuses = enum.InitFor[Status, struct {
	enum.Struct[Status]
	Pending Status `enum:"pending,待处理"`
	Active  Status `enum:"active,活跃"`
}]()

// Shadows locally — must NOT be flagged.
func shadow() {
	Statuses := "local"
	Statuses = Statuses + "!"
	_ = Statuses
}

// Writes to the package-level enum — MUST be flagged.
func realWrite() {
	Statuses.Pending = "hacked"
}
