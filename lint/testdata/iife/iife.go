// Package iife is a testdata fixture: the enum is built inside an IIFE
// so AddExt can run at construction. The lint tool must look through the
// IIFE, recognize the enum variable, and flag writes to it.
package iife

import "github.com/donnol/enum"

type Status string

type Statuses struct {
	enum.Struct[Status]
	Pending Status `enum:"pending,待处理"`
	Active  Status `enum:"active,活跃"`
}

var All = func() Statuses {
	c := enum.InitFor[Status, Statuses]()
	c.AddExt("Pending", "rel", "product.id")
	return c
}()

func use() Status {
	return All.Pending
}

func badWrite() {
	All.Active = "hacked"
}

// NotEnum wraps a plain value in an IIFE without calling enum.InitFor —
// its var must not be tracked.
var NotEnum = func() int { return 1 }()

func writeNotEnum() {
	NotEnum = 2
}
