// Package good is a testdata fixture: defines an enum and only reads
// from it. The lint tool should report zero violations here.
package good

import "github.com/donnol/enum"

type Status string

var Statuses = enum.InitFor[Status, struct {
	enum.Struct[Status]
	Pending Status `enum:"pending,待处理"`
	Active  Status `enum:"active,活跃"`
}]()

func use() Status {
	return Statuses.Pending
}

func check(s Status) bool {
	return s == Statuses.Active
}
