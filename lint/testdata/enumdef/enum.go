// Package enumdef is a testdata fixture that defines an enum for
// cross-package write-detection tests.
package enumdef

import "github.com/donnol/enum"

type Status string

var Statuses = enum.InitFor[Status, struct {
	enum.Struct[Status]
	Pending Status `enum:"pending,待处理"`
	Active  Status `enum:"active,活跃"`
}]()
