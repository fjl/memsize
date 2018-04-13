package memsize

import (
	"bytes"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"unsafe"
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
		ctx.s.BitmapSize = ctx.seen.size()
		ctx.s.BitmapUtilization = ctx.seen.utilization()
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
	// Internal stats (for debugging)
	BitmapSize        uintptr
	BitmapUtilization float32
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
		fmt.Fprintln(buf, line.name, strings.Repeat(" ", maxwidth-len(line.name)), HumanSize(line.total))
	}
	return buf.String()
}

// addValue is called during scan and adds the memory of given object.
func (s *Sizes) addValue(root string, v reflect.Value, size uintptr) {
	s.ByRoot[root] += size
	rs := s.ByType[v.Type()]
	if rs == nil {
		rs = &TypeSize{ByRoot: make(map[string]uintptr)}
		s.ByType[v.Type()] = rs
	}
	rs.ByRoot[root] += size
	rs.Total += size
}

type context struct {
	// We track previously seen objects to prevent infinite loops when scanning cycles, and
	// to prevent scanning objects more than once. This is done in two ways:
	//
	// - seen holds memory spans that have been scanned already. It prevents
	//   counting objects more than once.
	// - visiting holds pointers on the scan stack. It prevents going into an
	//   infinite loop for cyclic data.
	seen     *bitmap
	visiting map[address]reflect.Type

	tc      typCache
	s       *Sizes
	curRoot string
}

func newContext() *context {
	return &context{
		seen:     newBitmap(),
		visiting: make(map[address]reflect.Type),
		tc:       make(typCache),
		s:        newSizes(),
	}
}

// scan walks all objects below v, determining their size. All scan* functions return the
// amount of 'extra' memory (e.g. slice data) that is referenced by the object.
func (c *context) scan(addr address, v reflect.Value, add bool) (extraSize uintptr) {
	size := v.Type().Size()
	if addr.valid() {
		// Skip this value if it was scanned earlier.
		if c.seen.isMarked(uintptr(addr)) {
			return 0
		}
		// Also skip if it is being scanned already.
		// Problem: when scanning structs/arrays, the first field/element has the base
		// address and would be skipped. To fix this, we track the type for each seen
		// object and rescan if the addr is of different type. This works because the
		// type of the field/element can never be the same type as the containing
		// struct/array.
		if typ, ok := c.visiting[addr]; ok && isEqualOrPointerTo(v.Type(), typ) {
			return 0
		}
		c.visiting[addr] = v.Type()
	}
	extra := uintptr(0)
	if c.tc.needScan(v.Type()) {
		extra = c.scanContent(addr, v)
	}
	if addr.valid() {
		delete(c.visiting, addr)
		c.seen.markRange(uintptr(addr), size)
	}
	if add {
		size += extra
		c.s.addValue(c.curRoot, v, size)
	}
	return extra
}

func (c *context) scanContent(addr address, v reflect.Value) uintptr {
	switch v.Kind() {
	case reflect.Array:
		return c.scanArray(addr, v)
	case reflect.Chan:
		return c.scanChan(v)
	case reflect.Func:
		// can't do anything here
		return 0
	case reflect.Interface:
		return c.scanInterface(v)
	case reflect.Map:
		return c.scanMap(v)
	case reflect.Ptr:
		if !v.IsNil() {
			c.scan(address(v.Pointer()), v.Elem(), true)
		}
		return 0
	case reflect.Slice:
		return c.scanSlice(v)
	case reflect.String:
		return uintptr(v.Len())
	case reflect.Struct:
		return c.scanStruct(addr, v)
	default:
		unhandledKind(v.Kind())
		return 0
	}
}

func (c *context) scanChan(v reflect.Value) uintptr {
	etyp := v.Type().Elem()
	extra := uintptr(0)
	if c.tc.needScan(etyp) {
		// Scan the channel buffer. This is unsafe but doesn't race because
		// the world is stopped during scan.
		hchan := unsafe.Pointer(v.Pointer())
		for i := uint(0); i < uint(v.Cap()); i++ {
			addr := chanbuf(hchan, i)
			elem := reflect.NewAt(etyp, addr).Elem()
			extra += c.scan(address(addr), elem, false)
		}
	}
	return uintptr(v.Cap())*etyp.Size() + extra
}

func (c *context) scanStruct(base address, v reflect.Value) uintptr {
	extra := uintptr(0)
	for i := 0; i < v.NumField(); i++ {
		addr := base.addOffset(v.Type().Field(i).Offset)
		extra += c.scan(addr, v.Field(i), false)
	}
	return extra
}

func (c *context) scanArray(addr address, v reflect.Value) uintptr {
	_, extra := c.scanArrayMem(addr, v)
	return extra
}

func (c *context) scanSlice(v reflect.Value) uintptr {
	slice := v.Slice(0, v.Cap())
	count, extra := c.scanArrayMem(address(v.Pointer()), slice)
	return extra + uintptr(count)*v.Type().Elem().Size()
}

func (c *context) scanArrayMem(base address, slice reflect.Value) (count int, extra uintptr) {
	var (
		addr  = base
		esize = slice.Type().Elem().Size()
		escan = c.tc.needScan(slice.Type().Elem())
	)
	// Check whether the backing array is already tracked. If it is, scan only the
	// previously unscanned portion of the array to avoid counting overlapping slices
	// more than once.
	for i := 0; i < slice.Len(); i++ {
		if !c.seen.isMarked(uintptr(addr)) {
			if escan {
				extra += c.scan(addr, slice.Index(i), false)
			}
			c.seen.markRange(uintptr(addr), esize)
			count++
		}
		addr = addr.addOffset(esize)
	}
	return count, extra
}

func (c *context) scanMap(v reflect.Value) uintptr {
	var (
		typ   = v.Type()
		len   = uintptr(v.Len())
		extra = uintptr(0)
	)
	if c.tc.needScan(typ.Key()) || c.tc.needScan(typ.Elem()) {
		for _, k := range v.MapKeys() {
			extra += c.scan(invalidAddr, k, false)
			extra += c.scan(invalidAddr, v.MapIndex(k), false)
		}
	}
	return len*typ.Key().Size() + len*typ.Elem().Size() + extra
}

func (c *context) scanInterface(v reflect.Value) uintptr {
	elem := v.Elem()
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
