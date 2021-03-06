package valuestore

import (
	"math"
	"testing"
)

func TestValuesMemRead(t *testing.T) {
	vs := New(nil)
	vm1 := &valuesMem{id: 1, vs: vs, values: []byte("0123456789abcdef")}
	vm2 := &valuesMem{id: 2, vs: vs, values: []byte("fedcba9876543210")}
	vs.valueLocBlocks = []valueLocBlock{nil, vm1, vm2}
	tsn := vm1.timestampnano()
	if tsn != math.MaxInt64 {
		t.Fatal(tsn)
	}
	ts, v, err := vm1.read(1, 2, 0x100, 5, 6, nil)
	if err != ErrNotFound {
		t.Fatal(err)
	}
	vm1.vs.vlm.Set(1, 2, 0x100, vm1.id, 5, 6, false)
	ts, v, err = vm1.read(1, 2, 0x100, 5, 6, nil)
	if err != nil {
		a, b, c, d := vm1.vs.vlm.Get(1, 2)
		t.Fatal(err, a, b, c, d)
	}
	if ts != 0x100 {
		t.Fatal(ts)
	}
	if string(v) != "56789a" {
		t.Fatal(string(v))
	}
	vm1.vs.vlm.Set(1, 2, 0x100|_TSB_DELETION, vm1.id, 5, 6, false)
	ts, v, err = vm1.read(1, 2, 0x100, 5, 6, nil)
	if err != ErrNotFound {
		t.Fatal(err)
	}
	vm1.vs.vlm.Set(1, 2, 0x200, vm2.id, 5, 6, false)
	ts, v, err = vm1.read(1, 2, 0x100, 5, 6, nil)
	if err != nil {
		t.Fatal(err)
	}
	if ts != 0x200 {
		t.Fatal(ts)
	}
	if string(v) != "a98765" {
		t.Fatal(string(v))
	}
}
