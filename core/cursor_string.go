package core

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// CursorString is a string indicating the Cursor.
type CursorString string

// Validate returns true, if it is valid. Otherwise, it returns false.
func (cs CursorString) Validate() bool {
	var dot, underscore, hyphen int
	for _, r := range []rune(cs) {
		if r == '.' && dot == 0 {
			dot++
		} else if r == '-' && dot == 0 {
			hyphen++
		} else if r == '_' {
			underscore++
			dot = 0
			hyphen = 0
		} else if !(r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}

// FormatCursorString returns a CursorString.
func FormatCursorString(args ...interface{}) CursorString {
	var str string

	// args
	var i64 int64
	i64t := reflect.TypeOf(i64)
	var ui64 uint64
	ui64t := reflect.TypeOf(ui64)
	var ti time.Time
	tit := reflect.TypeOf(ti)

	for i, arg := range args {
		if i > 0 {
			str += "_"
		}
		str += (func() string {
			if arg == nil {
				return ""
			}

			v := reflect.ValueOf(arg)
			if v.Kind() == reflect.Ptr && v.IsNil() {
				return ""
			}

			v = reflect.Indirect(v)
			if v.Type().ConvertibleTo(i64t) {
				return strconv.FormatInt(v.Convert(i64t).Interface().(int64), 10)
			} else if v.Type().ConvertibleTo(ui64t) {
				return strconv.FormatUint(v.Convert(ui64t).Interface().(uint64), 10)
			} else if v.Type().ConvertibleTo(tit) {
				t := v.Convert(tit).Interface().(time.Time)
				s := strconv.FormatInt(t.Unix(), 10)
				nano := strconv.Itoa(t.Nanosecond())
				s += "." + strings.Repeat("0", 9-len(nano)) + nano
				s = strings.TrimRight(s, "0")
				s = strings.TrimRight(s, ".")
				return s
			}
			panic(fmt.Sprintf("Unsupported type arg specified: arg = %v", arg))
		})()
	}
	return CursorString(str)
}
