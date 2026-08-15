// Package noenum is a testdata fixture: no enum package is referenced at
// all — neither imported nor declaring InitFor. The lint tool must warn
// that enum write checks are disabled instead of silently passing.
package noenum

var X = 1

func write() {
	X = 2
}
