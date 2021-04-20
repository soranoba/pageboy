package core

import (
	"reflect"
	"strconv"
	"strings"
	"time"
)

// CursorSegment is a result of parsing each element of cursor.
type CursorSegment struct {
	integer int64
	nano    int64
	isNil   bool
}

// IsNil returns true if it have nil value. Otherwise, it returns false.
func (seg CursorSegment) IsNil() bool {
	return seg.isNil
}

// Int64 returns converted to integer.
func (seg CursorSegment) Int64() int64 {
	return seg.integer
}

// Int64Ptr returns converted to pointer of integer.
func (seg CursorSegment) Int64Ptr() *int64 {
	if seg.isNil {
		return nil
	}
	i := seg.integer
	return &i
}

// Bool returns converted to bool.
func (seg CursorSegment) Bool() bool {
	if seg.integer > 0 {
		return true
	}
	return false
}

// BoolPtr returns converted to pointer of bool.
func (seg CursorSegment) BoolPtr() *bool {
	if seg.isNil {
		return nil
	}
	b := seg.Bool()
	return &b
}

// Time returns converted to time.
func (seg CursorSegment) Time() *time.Time {
	if seg.isNil {
		return nil
	}
	t := time.Unix(seg.integer, seg.nano)
	return &t
}

// Interface returns converted to the type of the specified column.
func (seg CursorSegment) Interface(ty reflect.Type, column string) interface{} {
	if ty.Kind() != reflect.Struct {
		panic("model must be struct")
	}

	field, ok := ty.FieldByName(column)
	if !ok {
		return seg.Int64()
	}

	if field.Type == reflect.TypeOf(time.Time{}) ||
		field.Type == reflect.TypeOf(new(time.Time)) {
		return seg.Time()
	}

	switch field.Type.Kind() {
	case reflect.Ptr:
		if field.Type.Elem().Kind() == reflect.Bool {
			return seg.BoolPtr()
		}
		return seg.Int64Ptr()
	case reflect.Bool:
		return seg.Bool()
	default:
		return seg.Int64()
	}
}

// CursorSegments is slice of CursorSegment.
type CursorSegments []CursorSegment

// Interface returns slice of interface that converted to types of specified columns.
func (segs CursorSegments) Interface(ty reflect.Type, columns ...string) []interface{} {
	if len(segs) != len(columns) {
		panic("invalid number of columns")
	}

	results := make([]interface{}, len(columns))
	for i, column := range columns {
		results[i] = segs[i].Interface(ty, column)
	}
	return results
}

// NewCursorSegments create a CursorSegments from CursorString,
func NewCursorSegments(str CursorString) CursorSegments {
	parts := strings.Split(string(str), "_")

	if len(parts) == 0 {
		panic("invalid cursor")
	}

	args := make([]CursorSegment, len(parts))

	for i, part := range parts {
		if part == "" {
			args[i] = CursorSegment{isNil: true}
			continue
		}

		numberParts := strings.Split(part, ".")
		integer, err := strconv.ParseInt(numberParts[0], 10, 64)
		if err != nil {
			panic("invalid cursor")
		}
		nano := int64(0)
		if len(numberParts) > 1 {
			numberParts[1] += strings.Repeat("0", 9-len(numberParts[1]))
			numberParts[1] = numberParts[1][0:9]
			nano, err = strconv.ParseInt(numberParts[1], 10, 64)
			if err != nil {
				panic("invalid cursor")
			}
		}

		args[i] = CursorSegment{integer: integer, nano: nano}
	}

	return args
}
