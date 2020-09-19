package core

import (
	"fmt"
	"testing"
	"time"
)

func TestNewCursorSegments(t *testing.T) {
	format := "2006-01-02T15:04:05.9999"
	ti, err := time.Parse(format, "2020-04-01T02:03:04.0250")
	if err != nil {
		t.Error(err)
	}

	ga := NewCursorSegments("1585706584.025")
	assertEqual(t, len(ga), 1)
	assertEqual(t, ga[0].Time().UnixNano(), ti.UnixNano())

	ga = NewCursorSegments("1585706584.025_20")
	assertEqual(t, len(ga), 2)
	assertEqual(t, ga[0].Time().UnixNano(), ti.UnixNano())
	assertEqual(t, ga[1].Int64(), int64(20))
	assertEqual(t, *ga[1].Int64Ptr(), int64(20))

	ti, err = time.Parse(format, "2020-04-01T02:03:04")
	if err != nil {
		t.Error(err)
	}

	ga = NewCursorSegments("1585706584")
	assertEqual(t, len(ga), 1)
	assertEqual(t, ga[0].Int64(), int64(1585706584))
	assertEqual(t, *ga[0].Int64Ptr(), int64(1585706584))

	ga = NewCursorSegments("1585706584_20")
	assertEqual(t, len(ga), 2)
	assertEqual(t, ga[0].Time().UnixNano(), ti.UnixNano())
	assertEqual(t, ga[1].Int64(), int64(20))

	ga = NewCursorSegments("_1")
	assertEqual(t, len(ga), 2)
	assertEqual(t, ga[0].Time(), (*time.Time)(nil))
	assertEqual(t, ga[0].Int64(), int64(0))
	assertEqual(t, ga[0].Int64Ptr(), (*int64)(nil))
	assertEqual(t, ga[1].Int64(), int64(1))

	ga = NewCursorSegments("_1__2")
	assertEqual(t, len(ga), 4)
	assertEqual(t, ga[0].Time(), (*time.Time)(nil))
	assertEqual(t, ga[1].Int64(), int64(1))
	assertEqual(t, ga[2].IsNil(), true)
	assertEqual(t, ga[3].Int64(), int64(2))
}

func ExampleNewCursorSegments() {
	segments := NewCursorSegments("1585706584.025_20")
	fmt.Println(segments[0].Time().UTC().String())
	fmt.Println(segments[1].Int64())

	// Output:
	// 2020-04-01 02:03:04.025 +0000 UTC
	// 20
}
