// Package namedtype is a testdata fixture: the enum struct is a named
// type (not an inline anonymous struct). The lint tool must still
// recognize the enum variable and flag writes to it.
package namedtype

import "github.com/donnol/enum"

type Status string

type Statuses struct {
	enum.Struct[Status]
	Pending Status `enum:"pending,待处理"`
	Active  Status `enum:"active,活跃"`
}

var All = enum.InitFor[Status, Statuses]()

func use() Status {
	return All.Pending
}

func badWrite() {
	All.Active = "hacked"
}

// NotAnEnum is a named struct but NOT built via enum.InitFor — its var
// must not be tracked, so the write below is legitimate.
type plain struct{ X int }

var NotAnEnum = plain{}

func writePlain() {
	NotAnEnum.X = 1
}
