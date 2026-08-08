// Package fakeenum is a testdata fixture that declares an InitFor-like
// generic function. The lint tool's A1 discovery should recognize this
// package as the enum provider and match qualified InitFor calls to it.
package fakeenum

type EB interface{ ~int | ~string }

func InitFor[T EB, R any]() R {
	var z R
	return z
}
