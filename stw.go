package memsize

import "unsafe"

var _ = unsafe.Pointer(nil)

//go:linkname stopTheWorld runtime.stopTheWorld
func stopTheWorld(reason string)

//go:linkname startTheWorld runtime.startTheWorld
func startTheWorld()
