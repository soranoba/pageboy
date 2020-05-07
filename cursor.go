package magion

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
)

// Cursor can to get a specific range of records from DB in time order.
//
// When limit is smaller than or equal to 0, the validation will fail.
// You should set the initial values and then read it from query or json.
//
//   cursor := &Cursor{Limit: 10}
//   ctx.Bind(cursor)
//
type Cursor struct {
	Before string `json:"before" query:"before"`
	After  string `json:"after" query:"after"`
	Limit  int    `json:"limit" query:"limit"`

	nextBefore *string
	nextAfter  *string
	limit      int64
	hasBefore  bool
}

func init() {
	gorm.DefaultCallback.Query().Before("gorm:query").
		Register("magion:cursor:before_query", cursorHandleBeforeQuery)
	gorm.DefaultCallback.Query().After("gorm:query").
		Register("magion:cursor:after_query", cursorHandleAfterQuery)
	gorm.DefaultCallback.Query().
		Register("magion:cursor:handle_query", cursorHandleQuery)
}

// Validate returns true when the Cursor is valid. Otherwise, it returns false.
// If you execute Paginate with an invalid value, panic may occur.
func (cursor *Cursor) Validate() error {
	if cursor.Before != "" && cursor.After != "" {
		return errors.New("Both of before and after cannot be specified")
	}
	if cursor.Before != "" && !ValidateCursorString(cursor.Before) {
		return errors.New("The before parameter is invalid")
	}
	if cursor.After != "" && !ValidateCursorString(cursor.After) {
		return errors.New("The after parameter is invalid")
	}
	if cursor.Limit < 1 {
		return errors.New("The limit parameter is invalid")
	}

	return nil
}

// GetNextAfter returns a query to access after the current position if it exists some records.
func (cursor *Cursor) GetNextAfter() *string {
	return cursor.nextAfter
}

// GetNextBefore returns a query to access before the current position if it exists some records.
func (cursor *Cursor) GetNextBefore() *string {
	return cursor.nextBefore
}

// Paginate is a scope for the gorm.
//
// Example:
//
//   db.Scopes(cursor.Paginate("CreatedAt", "ID")).Find(&models)
//
func (cursor *Cursor) Paginate(timeColumn string, columns ...string) func(db *gorm.DB) *gorm.DB {
	columns = append([]string{timeColumn}, columns...)

	return func(db *gorm.DB) *gorm.DB {
		db = db.New().
			InstantSet("magion:columns", columns).
			InstantSet("magion:cursor", cursor)

		db = (func() *gorm.DB {
			if cursor.Before == "" && cursor.After == "" {
				return db.Order(CompositeOrder("DESC", columns...))
			} else if cursor.Before != "" {
				t, args := ParseCursorString(cursor.Before)
				args = append([]interface{}{t}, args...)
				return db.
					Scopes(CompositeSortScopeFunc("<", columns...)(args...)).
					Order(CompositeOrder("DESC", columns...))
			} else if cursor.After != "" {
				t, args := ParseCursorString(cursor.After)
				args = append([]interface{}{t}, args...)
				return db.
					Scopes(CompositeSortScopeFunc(">", columns...)(args...)).
					Order(CompositeOrder("ASC", columns...))
			} else {
				panic("invalid cursor")
			}
		})()

		return db.Limit(cursor.Limit)
	}
}

// FormatCursorString returns a string for Cursor from time and integers.
func FormatCursorString(t time.Time, args ...interface{}) string {
	// time
	str := strconv.FormatInt(t.Unix(), 10) + "." + strconv.Itoa(t.Nanosecond())
	str = strings.TrimRight(str, "0")
	str = strings.TrimRight(str, ".")

	// args
	var i64 int64
	i64t := reflect.TypeOf(i64)
	var ui64 uint64
	ui64t := reflect.TypeOf(ui64)

	for _, arg := range args {
		str += "_" + (func() string {
			v := reflect.Indirect(reflect.ValueOf(arg))
			if v.Type().ConvertibleTo(i64t) {
				return strconv.FormatInt(v.Convert(i64t).Interface().(int64), 10)
			} else if v.Type().ConvertibleTo(ui64t) {
				return strconv.FormatUint(v.Convert(ui64t).Interface().(uint64), 10)
			}
			panic(fmt.Sprintf("Unsupported type arg specified: arg = %v", arg))
		})()
	}
	return str
}

// ValidateCursorString returns true, if an argument is valid a cursor string. Otherwise, it returns false.
func ValidateCursorString(str string) bool {
	var dot, underscore, hyphen int
	for _, r := range []rune(str) {
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

// ParseCursorString parses a string for cursor to a time and integers.
func ParseCursorString(str string) (time.Time, []interface{}) {
	parts := strings.Split(str, "_")

	if len(parts) == 0 {
		panic("invalid cursor")
	}

	unix, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		panic("invalid cursor")
	}
	t := unixToTime(unix)
	args := make([]interface{}, len(parts)-1)

	for i, part := range parts[1:] {
		v, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			panic("invalid cursor")
		}
		args[i] = v
	}

	return t, args
}

func getCursor(scope *gorm.Scope) (*Cursor, bool) {
	value, ok := scope.Get("magion:cursor")
	if !ok {
		return nil, false
	}
	cursor, ok := value.(*Cursor)
	if !ok {
		return nil, false
	}
	return cursor, true
}

func getColumns(scope *gorm.Scope) ([]string, bool) {
	value, ok := scope.Get("magion:columns")
	if !ok {
		return nil, false
	}
	columns, ok := value.([]string)
	if !ok {
		return nil, false
	}
	return columns, true
}

func cursorHandleBeforeQuery(scope *gorm.Scope) {
	cursor, ok := getCursor(scope)
	if !ok {
		return
	}

	re := regexp.MustCompile(`LIMIT\s+([0-9]+)`)

	matches := re.FindStringSubmatch(scope.DB().NewScope(scope.DB().Value).CombinedConditionSql())
	if len(matches) == 2 {
		limitStr := matches[1]
		limit, err := strconv.ParseInt(limitStr, 10, 64)
		if err == nil {
			cursor.limit = limit
			if cursor.After == "" {
				scope.Search.Limit(limit + 1)
			}
			return
		}
	}
	cursor.limit = -1
}

func cursorHandleAfterQuery(scope *gorm.Scope) {
	cursor, ok := getCursor(scope)
	if !ok {
		return
	}

	cursor.hasBefore = false
	if cursor.limit == -1 {
		return
	}

	results := scope.IndirectValue()
	if !(results.Kind() == reflect.Array || results.Kind() == reflect.Slice) {
		return
	}
	if cursor.limit+1 == int64(results.Len()) {
		cursor.hasBefore = true
		results.Set(results.Slice(0, results.Len()-1))
	}
	if cursor.After != "" {
		cursor.hasBefore = true
		for i := 0; i < results.Len()/2; i++ {
			s1 := results.Index(i)
			s2 := results.Index(results.Len() - 1 - i)
			v1 := s1.Interface()
			v2 := s2.Interface()
			s2.Set(reflect.ValueOf(v1))
			s1.Set(reflect.ValueOf(v2))
		}
	}
}

func cursorHandleQuery(scope *gorm.Scope) {
	cursor, ok := getCursor(scope)
	if !ok {
		return
	}
	columns, ok := getColumns(scope)
	if !ok {
		return
	}

	cursor.nextBefore = nil
	cursor.nextAfter = nil

	if scope.HasError() {
		return
	}
	results := scope.IndirectValue()
	if !(results.Kind() == reflect.Array || results.Kind() == reflect.Slice) {
		return
	}

	length := results.Len()
	if length == 0 {
		if cursor.After != "" {
			cursor.nextAfter = &cursor.After
			var emptyStr string
			cursor.nextBefore = &emptyStr
		} else {
			cursor.nextAfter = getCursorStringFromColumns(reflect.New(results.Type().Elem()), columns...)
		}
	} else {
		cursor.nextAfter = getCursorStringFromColumns(results.Index(0), columns...)
		if cursor.hasBefore {
			cursor.nextBefore = getCursorStringFromColumns(results.Index(length-1), columns...)
		}
	}
}

func getCursorStringFromColumns(value reflect.Value, columns ...string) *string {
	value = reflect.Indirect(value)
	if !(value.Kind() == reflect.Struct && value.Kind() == reflect.Struct) {
		return nil
	}
	if len(columns) == 0 {
		return nil
	}

	timeValue := value.FieldByName(columns[0])
	timeValue = reflect.Indirect(timeValue)
	if !(timeValue.Kind() == reflect.Struct && timeValue.CanInterface()) {
		return nil
	}

	t, ok := timeValue.Interface().(time.Time)
	if !ok {
		return nil
	}

	args := make([]interface{}, len(columns)-1)
	for i, column := range columns[1:] {
		argValue := value.FieldByName(column)
		args[i] = argValue.Interface()
	}

	str := FormatCursorString(t, args...)
	return &str
}
