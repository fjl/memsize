package memsize_test

import (
	"testing"
	"unsafe"

	"github.com/fjl/memsize"
)

const (
	sizeofSlice     = unsafe.Sizeof([]byte{})
	sizeofMap       = unsafe.Sizeof(map[string]string{})
	sizeofInterface = unsafe.Sizeof((interface{})(nil))
	sizeofString    = unsafe.Sizeof("")
)

type (
	struct16 struct {
		x, y uint64
	}
	struct32ptr struct {
		x   uint32
		cld *struct32ptr
	}
	struct64array struct {
		array64
	}
	structslice struct {
		s []uint32
	}
	structstring struct {
		s string
	}
	array64 [64]byte
)

func TestTotal(t *testing.T) {
	tests := []struct {
		name string
		v    interface{}
		want uintptr
	}{
		{
			name: "struct16",
			v:    &struct16{},
			want: 16,
		},
		{
			name: "struct32ptr_nil",
			v:    &struct32ptr{},
			want: 16,
		},
		{
			name: "struct32ptr",
			v:    &struct32ptr{cld: &struct32ptr{}},
			want: 32,
		},
		{
			name: "struct32ptr_loop",
			v: func() *struct32ptr {
				v := &struct32ptr{}
				v.cld = v
				return v
			}(),
			want: 16,
		},
		{
			name: "struct64array",
			v:    &struct64array{},
			want: 64,
		},
		{
			name: "structslice",
			v:    &structslice{s: []uint32{1, 2, 3}},
			want: sizeofSlice + 3*4,
		},
		{
			name: "array64",
			v:    &array64{},
			want: 64,
		},
		{
			name: "byteslice",
			v:    &[]byte{1, 2, 3},
			want: 27,
		},
		{
			name: "slice3_ptrval",
			v:    &[]*struct16{{}, {}, {}},
			want: 96,
		},
		{
			name: "map3",
			v:    &map[uint64]uint64{1: 1, 2: 2, 3: 3},
			want: 56,
		},
		{
			name: "map3_ptrval",
			v:    &map[uint64]*struct16{1: {}, 2: {}, 3: {}},
			want: 104,
		},
		{
			name: "map3_ptrkey",
			v:    &map[*struct16]uint64{{x: 1}: 1, {x: 2}: 2, {x: 3}: 3},
			want: 104,
		},
		{
			name: "pointerpointer",
			v: func() **uint64 {
				i := uint64(0)
				p := &i
				return &p
			}(),
			want: 16,
		},
		{
			name: "structstring",
			v:    &structstring{"123"},
			want: sizeofString + 3,
		},
		{
			name: "slices_samearray",
			v: func() *[3][]byte {
				backarray := [64]byte{}
				return &[3][]byte{
					backarray[16:],
					backarray[4:16],
					backarray[0:4],
				}
			}(),
			want: 3*sizeofSlice + 64,
		},
		{
			name: "slices_nil",
			v: func() *[2][]byte {
				return &[2][]byte{nil, nil}
			}(),
			want: 2 * sizeofSlice,
		},
		{
			name: "slices_overlap_total",
			v: func() *[2][]byte {
				backarray := [32]byte{}
				return &[2][]byte{backarray[:], backarray[:]}
			}(),
			want: 2*sizeofSlice + 32,
		},
		{
			name: "slices_overlap",
			v: func() *[4][]uint16 {
				backarray := [32]uint16{}
				return &[4][]uint16{
					backarray[2:4],
					backarray[10:12],
					backarray[20:25],
					backarray[:],
				}
			}(),
			want: 4*sizeofSlice + 32*2,
		},
		{
			name: "slices_overlap_array",
			v: func() *struct {
				a [32]byte
				s [2][]byte
			} {
				v := struct {
					a [32]byte
					s [2][]byte
				}{}
				v.s[0] = v.a[2:4]
				v.s[1] = v.a[5:8]
				return &v
			}(),
			want: 32 + 2*sizeofSlice,
		},
		{
			name: "interface",
			v:    &[2]interface{}{uint64(0), &struct16{}},
			want: 2*sizeofInterface + 8 + 16,
		},
		{
			name: "interface_nil",
			v:    &[2]interface{}{nil, nil},
			want: 2 * sizeofInterface,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			var rs memsize.RootSet
			rs.Add("test", test.v)
			size := rs.Scan()
			if size.Total() != test.want {
				t.Errorf("total=%d, want %d", size.Total(), test.want)
				t.Logf("\n%s", size.Report())

			}
		})
	}
}
