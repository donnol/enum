// Package singleidx is a testdata fixture: calls enum.InitFor with a
// single type arg (no struct). This exercises the *ast.IndexExpr branch
// in findEnumVars. Since there's no struct type arg, the call is
// correctly NOT tracked as an enum.
package singleidx

import "github.com/donnol/enum"

type Status string

// Single type arg — parsed as *ast.IndexExpr (not IndexListExpr).
// findEnumVars enters the IndexExpr case but skips it (len(indices) < 2).
// Not an enum, so writes to NotAnEnum must NOT be flagged.
var NotAnEnum = enum.InitFor[Status]()

func write() {
	NotAnEnum = "x" // should NOT be flagged — NotAnEnum is not tracked
}
