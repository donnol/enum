// Package usefake is a testdata fixture that imports the fakeenum
// package and defines an enum via fakeenum.InitFor. With A1 discovery,
// writes to it must be flagged WITHOUT an explicit -enum-pkg flag.
package usefake

import "github.com/donnol/enum/lint/testdata/discover/fakeenum"

type S string

var Statuses = fakeenum.InitFor[S, struct {
	Pending S `enum:"pending,待处理"`
	Active  S `enum:"active,活跃"`
}]()

func badWrite() {
	Statuses.Pending = "hacked"
}
