package memsize

import (
	"fmt"
	"reflect"
)

// address is a memory location.
//
// Code dealing with uintptr is oblivious to the zero address.
// Code dealing with address is not: it treats the zero address
// as invalid. Offsetting an invalid address doesn't do anything.
//
// This distinction is useful because there are objects that we can't
// get the pointer to.
type address uintptr

const invalidAddr = address(0)

func (a address) valid() bool {
	return a != 0
}

func (a address) addOffset(off uintptr) address {
	if !a.valid() {
		return invalidAddr
	}
	return a + address(off)
}

func (a address) String() string {
	return fmt.Sprintf("%#x", uintptr(a))
}

type typCache map[reflect.Type]typInfo

type typInfo struct {
	isPointer bool
	needScan  bool
}

func (tc *typCache) isPointer(typ reflect.Type) bool {
	return tc.info(typ).isPointer
}

func (tc *typCache) needScan(typ reflect.Type) bool {
	return tc.info(typ).needScan
}

func (tc *typCache) info(typ reflect.Type) typInfo {
	if info, ok := (*tc)[typ]; ok {
		return info
	}
	info := tc.makeInfo(typ)
	(*tc)[typ] = info
	return info
}

func (tc *typCache) makeInfo(typ reflect.Type) typInfo {
	var ti typInfo
	ti.isPointer = isPointer(typ)
	ti.needScan = ti.isPointer
	k := typ.Kind()
	if k == reflect.Array {
		ti.needScan = tc.needScan(typ.Elem())
	} else if k >= reflect.Chan && k <= reflect.Struct {
		ti.needScan = true
	}
	return ti
}

func isPointer(typ reflect.Type) bool {
	k := typ.Kind()
	switch {
	case k <= reflect.Complex128:
		return false
	case k == reflect.Array:
		return false
	case k >= reflect.Chan && k <= reflect.String:
		return true
	case k == reflect.Struct || k == reflect.UnsafePointer:
		return false
	default:
		unhandledKind(k)
		return false
	}
}

func unhandledKind(k reflect.Kind) {
	panic("unhandled kind " + k.String())
}

func humanSize(bytes uintptr) string {
	switch {
	case bytes < 1024:
		return fmt.Sprintf("%d B", bytes)
	case bytes < 1024*1024:
		return fmt.Sprintf("%.3f KB", float64(bytes)/1024)
	default:
		return fmt.Sprintf("%.3f MB", float64(bytes)/1024/1024)
	}
}
