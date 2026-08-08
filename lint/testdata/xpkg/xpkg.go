// Package xpkg is a testdata fixture that writes to an enum defined in
// another testdata package (enumdef). Tests cross-package detection.
package xpkg

import "github.com/donnol/enum/lint/testdata/enumdef"

func badWrite() {
	enumdef.Statuses.Pending = "hacked"
}

func aliasedWrite() {
	// Custom alias import would be a separate file; here we use the
	// default alias (package name = enumdef).
	_ = enumdef.Statuses.Active // read — should NOT be flagged
}
