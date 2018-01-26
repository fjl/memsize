package memsize

import (
	"bytes"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// RootSet holds roots to scan.
type RootSet struct {
	roots map[string]reflect.Value
}

// Add adds a root to scan. The value must be a non-nil pointer to any value.
func (g *RootSet) Add(name string, obj interface{}) {
	if g.roots == nil {
		g.roots = make(map[string]reflect.Value)
	}
	rv := reflect.ValueOf(obj)
	if rv.Kind() != reflect.Ptr {
		panic("root must be pointer")
	}
	g.roots[name] = rv
}

// Roots returns all registered root names.
func (g *RootSet) Roots() []string {
	names := make([]string, 0, len(g.roots))
	for name := range g.roots {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Scan traverses all objects reachable from the current roots and counts how much memory
// is used per-type and per-root.
func (g *RootSet) Scan() Sizes {
	return g.scan(g.roots)
}

// ScanRoot scans a single root.
func (g *RootSet) ScanRoot(name string) Sizes {
	singleRoot := map[string]reflect.Value{name: g.roots[name]}
	if _, ok := g.roots[name]; !ok {
		panic("memsize: ScanRoot called with unregistered root name")
	}
	return g.scan(singleRoot)
}

func (g *RootSet) scan(roots map[string]reflect.Value) Sizes {
	stopTheWorld("memsize scan")
	defer startTheWorld()

	ctx := newContext()
	for name, root := range roots {
		ctx.curRoot = name
		ctx.scan(invalidAddr, root, false)
	}
	return *ctx.s
}

// Scan is a shorthand for scanning a single unnamed root.
func Scan(root interface{}) Sizes {
	var g RootSet
	g.Add("", root)
	return g.Scan()
}

// Sizes is the result of a scan.
type Sizes struct {
	ByRoot map[string]uintptr
	ByType map[reflect.Type]*TypeSize
}

type TypeSize struct {
	Total  uintptr
	ByRoot map[string]uintptr
}

func newSizes() *Sizes {
	return &Sizes{ByRoot: make(map[string]uintptr), ByType: make(map[reflect.Type]*TypeSize)}
}

// Total returns the total amount of memory across all roots.
func (s Sizes) Total() uintptr {
	var sum uintptr
	for _, r := range s.ByRoot {
		sum += r
	}
	return sum
}

// Report returns a human-readable report.
func (s Sizes) Report() string {
	// Make a type table
	type typLine struct {
		name  string
		total uintptr
	}
	tab := []typLine{{"TOTAL", s.Total()}}
	maxwidth := len(tab[0].name)
	for typ, s := range s.ByType {
		line := typLine{typ.String(), s.Total}
		tab = append(tab, line)
		if len(line.name) > maxwidth {
			maxwidth = len(line.name)
		}
	}
	sort.Slice(tab, func(i, j int) bool {
		return tab[i].total > tab[j].total
	})

	buf := new(bytes.Buffer)
	for _, line := range tab {
		fmt.Fprintln(buf, line.name, strings.Repeat(" ", maxwidth-len(line.name)), humanSize(line.total))
	}
	return buf.String()
}

// addValue is called during scan and adds the memory of given object.
func (s *Sizes) addValue(root string, obj *object) {
	s.ByRoot[root] += obj.size
	rs := s.ByType[obj.v.Type()]
	if rs == nil {
		rs = &TypeSize{ByRoot: make(map[string]uintptr)}
		s.ByType[obj.v.Type()] = rs
	}
	rs.ByRoot[root] += obj.size
	rs.Total += obj.size
}

type context struct {
	seen       map[uintptr]reflect.Type
	tc         typCache
	s          *Sizes
	curRoot    string
	backarrays memSpans
}

type object struct {
	v    reflect.Value
	size uintptr
}

func newContext() *context {
	return &context{
		seen: make(map[uintptr]reflect.Type),
		tc:   make(typCache),
		s:    newSizes(),
	}
}

// scan walks all objects below v, determining their size. All scan* functions return the
// amount of 'extra' memory (e.g. slice data) that is referenced by the object.
func (c *context) scan(addr address, v reflect.Value, add bool) (extraSize uintptr) {
	obj := &object{v: v, size: v.Type().Size()}
	extra := uintptr(0)
	if c.tc.needScan(v.Type()) {
		if addr.valid() {
			// Problem: when scanning structs/arrays, the first field/element has the base
			// address and would be skipped. To fix this, we track the type for each seen
			// object and rescan if the addr is of different type. This works because the
			// type of the field/element can never be the same type as the containing
			// struct/array.
			if typ, ok := c.seen[uintptr(addr)]; ok && isEqualOrPointerTo(v.Type(), typ) {
				// TODO: add it again if different root
				return
			}
			c.seen[uintptr(addr)] = v.Type()
		}
		extra = c.scanContent(addr, obj)
	}
	if add {
		obj.size += extra
		c.s.addValue(c.curRoot, obj)
	}
	return extra
}

func (c *context) scanContent(addr address, obj *object) uintptr {
	switch obj.v.Kind() {
	case reflect.Array:
		return c.scanArray(addr, obj)
	case reflect.Chan:
		return c.scanChan(obj)
	case reflect.Func:
		// can't do anything here
		return 0
	case reflect.Interface:
		return c.scanInterface(obj)
	case reflect.Map:
		return c.scanMap(obj)
	case reflect.Ptr:
		if !obj.v.IsNil() {
			return c.scan(address(obj.v.Pointer()), obj.v.Elem(), true)
		}
		return 0
	case reflect.Slice:
		return c.scanSlice(obj)
	case reflect.String:
		return uintptr(obj.v.Len())
	case reflect.Struct:
		return c.scanStruct(addr, obj)
	default:
		unhandledKind(obj.v.Kind())
		return 0
	}
}

func (c *context) scanChan(obj *object) uintptr {
	etyp := obj.v.Type().Elem()
	return uintptr(obj.v.Cap()) * etyp.Size()
}

func (c *context) scanStruct(base address, obj *object) uintptr {
	extra := uintptr(0)
	for i := 0; i < obj.v.NumField(); i++ {
		addr := base.addOffset(obj.v.Type().Field(i).Offset)
		extra += c.scan(addr, obj.v.Field(i), false)
	}
	return extra
}

func (c *context) scanArray(addr address, obj *object) uintptr {
	_, extra := c.scanArrayMem(addr, obj)
	return extra
}

func (c *context) scanSlice(obj *object) uintptr {
	count, extra := c.scanArrayMem(address(obj.v.Pointer()), obj)
	return extra + uintptr(count)*obj.v.Type().Elem().Size()
}

func (c *context) scanArrayMem(addr address, obj *object) (count int, extra uintptr) {
	var (
		esize   = obj.v.Type().Elem().Size()
		slice   = obj.v.Slice(0, obj.v.Cap())
		overlap memSpans
	)
	// Check whether the backing array is already tracked. If it is, scan only the
	// previously unscanned portion of the array to avoid counting overlapping slices
	// more than once.
	if addr.valid() {
		size := uintptr(obj.v.Cap()) * esize
		overlap = c.backarrays.insert(uintptr(addr), size)
	}
	for i := 0; i < slice.Len(); i++ {
		if !overlap.contains(addr) {
			extra += c.scan(addr, slice.Index(i), false)
			count++
		}
		addr = addr.addOffset(esize)
	}
	return count, extra
}

func (c *context) scanMap(obj *object) uintptr {
	var (
		typ   = obj.v.Type()
		len   = uintptr(obj.v.Len())
		extra = uintptr(0)
	)
	for _, k := range obj.v.MapKeys() {
		extra += c.scan(invalidAddr, k, false)
		extra += c.scan(invalidAddr, obj.v.MapIndex(k), false)
	}
	return len*typ.Key().Size() + len*typ.Elem().Size() + extra
}

func (c *context) scanInterface(obj *object) uintptr {
	elem := obj.v.Elem()
	if !elem.IsValid() {
		return 0 // nil interface
	}
	c.scan(invalidAddr, elem, false)
	if !c.tc.isPointer(elem.Type()) {
		// Account for non-pointer size of the value.
		return elem.Type().Size()
	}
	return 0
}
