package core

import (
	"fmt"
	"testing"
	"time"
)

func TestFormatCursorString(t *testing.T) {
	var flag bool = true
	var id1 uint = 20
	var id2 int = 10

	format := "2006-01-02T15:04:05.9999"
	ti, err := time.Parse(format, "2020-04-01T02:03:04.0250")
	if err != nil {
		t.Error(err)
	}
	assertEqual(t, FormatCursorString(&ti), CursorString("1585706584.025"))
	assertEqual(t, FormatCursorString(&ti, id1), CursorString("1585706584.025_20"))
	assertEqual(t, FormatCursorString(&ti, &id1), CursorString("1585706584.025_20"))
	assertEqual(t, FormatCursorString(&ti, id1, id2, flag, &flag), CursorString("1585706584.025_20_10_1_1"))

	ti, err = time.Parse(format, "2020-04-01T02:03:04")
	if err != nil {
		t.Error(err)
	}
	assertEqual(t, FormatCursorString(&ti), CursorString("1585706584"))
	assertEqual(t, FormatCursorString(&ti, id1), CursorString("1585706584_20"))
	assertEqual(t, FormatCursorString(&ti, &id1), CursorString("1585706584_20"))

	assertEqual(t, FormatCursorString(nil, nil), CursorString("_"))
	assertEqual(t, FormatCursorString(nil, nil, nil), CursorString("__"))
	assertEqual(t, FormatCursorString(nil, id1), CursorString("_20"))
}

func ExampleFormatCursorString() {
	format := "2006-01-02T15:04:05.9999"
	ti, _ := time.Parse(format, "2020-04-01T02:03:04")

	fmt.Println(FormatCursorString(ti, 1, 3.5))

	// Output:
	// 1585706584_1_3
}

func TestCursorString_Validate(t *testing.T) {
	assertEqual(t, CursorString("1585706584").Validate(), true)
	assertEqual(t, CursorString("1585706584.25").Validate(), true)
	assertEqual(t, CursorString("1585706584.250").Validate(), true)
	assertEqual(t, CursorString("1585706584.25_20").Validate(), true)
	assertEqual(t, CursorString("1585706584_20").Validate(), true)
	assertEqual(t, CursorString("1585706_584.25").Validate(), true)
	assertEqual(t, CursorString("1585706_584_25").Validate(), true)

	assertEqual(t, CursorString("15857065.84.25").Validate(), false)
	assertEqual(t, CursorString("1585706aa4.25").Validate(), false)
	assertEqual(t, CursorString("1585706584.2aa").Validate(), false)
}
