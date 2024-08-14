//go:build go1.23
// +build go1.23

package memsize

type stwReason uint8

const stwReadMemStats stwReason = 7

func stopTheWorld(reason stwReason) {
	panic("memsize is not supported with Go >= 1.23")
}
