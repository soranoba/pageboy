package pageboy

import (
	"errors"
	"fmt"
	"net/url"
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
	Order  Order  `json:"order" query:"order" enums:"asc,desc"`

	nextBefore string
	nextAfter  string
	limit      int64
	hasMore    bool
}

// CursorPagingUrls is for the user to access from the next cursor position.
type CursorPagingUrls struct {
	Before string `json:"before,omitempty"`
	After  string `json:"after,omitempty"`
}

func init() {
	gorm.DefaultCallback.Query().Before("gorm:query").
		Register("pageboy:cursor:before_query", cursorHandleBeforeQuery)
	gorm.DefaultCallback.Query().After("gorm:query").
		Register("pageboy:cursor:after_query", cursorHandleAfterQuery)
	gorm.DefaultCallback.Query().
		Register("pageboy:cursor:handle_query", cursorHandleQuery)
}

// NewDefaultCursor returns a default Cursor.
func NewDefaultCursor() *Cursor {
	return &Cursor{
		Limit: 10,
		Order: DESC,
	}
}

// Validate returns true when the Cursor is valid. Otherwise, it returns false.
// If you execute Paginate with an invalid value, panic may occur.
func (cursor *Cursor) Validate() error {
	if cursor.Before != "" && !ValidateCursorString(cursor.Before) {
		return errors.New("The before parameter is invalid")
	}
	if cursor.After != "" && !ValidateCursorString(cursor.After) {
		return errors.New("The after parameter is invalid")
	}
	if cursor.Limit < 1 {
		return errors.New("The limit parameter is invalid")
	}
	if cursor.Order != ASC && cursor.Order != DESC {
		return errors.New("The order parameter is invalid")
	}
	return nil
}

// GetNextAfter returns a query to access after the current position if it exists some records.
func (cursor *Cursor) GetNextAfter() string {
	return cursor.nextAfter
}

// GetNextBefore returns a query to access before the current position if it exists some records.
func (cursor *Cursor) GetNextBefore() string {
	return cursor.nextBefore
}

// BuildNextPagingUrls returns URLs for the user to access from the next cursor position.
func (cursor *Cursor) BuildNextPagingUrls(base *url.URL) *CursorPagingUrls {
	pagingUrls := &CursorPagingUrls{}

	if base == nil {
		return pagingUrls
	}

	baseUrl := *base

	(func() {
		// there are no older elements within the specified range.
		if cursor.Order == ASC {
			return
		}
		if cursor.Order == DESC && !cursor.hasMore {
			return
		}

		if cursor.nextBefore != "" {
			beforeUrl := baseUrl
			query := baseUrl.Query()
			query.Del("before")
			query.Add("before", cursor.nextBefore)
			beforeUrl.RawQuery = query.Encode()
			pagingUrls.Before = beforeUrl.String()
		}
	})()

	(func() {
		// there are no newer elements within the specified range.
		if cursor.Order == DESC {
			return
		}
		if cursor.Order == ASC && !cursor.hasMore {
			return
		}

		if cursor.nextAfter != "" {
			afterUrl := baseUrl
			query := baseUrl.Query()
			query.Del("after")
			query.Add("after", cursor.nextAfter)
			afterUrl.RawQuery = query.Encode()
			pagingUrls.After = afterUrl.String()
		}
	})()

	return pagingUrls
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
			InstantSet("pageboy:columns", columns).
			InstantSet("pageboy:cursor", cursor)

		if cursor.Before != "" {
			t, args := ParseCursorString(cursor.Before)
			args = append([]interface{}{t}, args...)
			db = db.Scopes(CompositeSortScopeFunc("<", columns...)(args...))
		}

		if cursor.After != "" {
			t, args := ParseCursorString(cursor.After)
			args = append([]interface{}{t}, args...)
			db = db.Scopes(CompositeSortScopeFunc(">", columns...)(args...))
		}

		return db.
			Order(CompositeOrder(cursor.Order, columns...)).
			Limit(cursor.Limit)
	}
}

// FormatCursorString returns a string for Cursor from time and integers.
func FormatCursorString(t *time.Time, args ...interface{}) string {
	var str string

	// time
	if t != nil {
		str = strconv.FormatInt(t.Unix(), 10) + "." + strconv.Itoa(t.Nanosecond())
		str = strings.TrimRight(str, "0")
		str = strings.TrimRight(str, ".")
	}

	// args
	var i64 int64
	i64t := reflect.TypeOf(i64)
	var ui64 uint64
	ui64t := reflect.TypeOf(ui64)

	for _, arg := range args {
		str += "_" + (func() string {
			if arg == nil {
				return ""
			}
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
func ParseCursorString(str string) (*time.Time, []interface{}) {
	parts := strings.Split(str, "_")

	if len(parts) == 0 {
		panic("invalid cursor")
	}

	t := (func() *time.Time {
		if parts[0] == "" {
			return nil
		}
		unix, err := strconv.ParseFloat(parts[0], 64)
		if err != nil {
			panic("invalid cursor")
		}
		return unixToTime(unix)
	})()
	args := make([]interface{}, len(parts)-1)

	for i, part := range parts[1:] {
		if part == "" {
			continue
		}
		v, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			panic("invalid cursor")
		}
		args[i] = v
	}

	return t, args
}

func getCursor(scope *gorm.Scope) (*Cursor, bool) {
	value, ok := scope.Get("pageboy:cursor")
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
	value, ok := scope.Get("pageboy:columns")
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
			scope.Search.Limit(limit + 1)
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

	cursor.hasMore = false
	if cursor.limit == -1 {
		return
	}

	results := scope.IndirectValue()
	if !(results.Kind() == reflect.Array || results.Kind() == reflect.Slice) {
		return
	}

	if cursor.limit+1 == int64(results.Len()) {
		cursor.hasMore = true
		results.Set(results.Slice(0, results.Len()-1))
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

	cursor.nextBefore = ""
	cursor.nextAfter = ""

	if scope.HasError() {
		return
	}
	results := scope.IndirectValue()
	if !(results.Kind() == reflect.Array || results.Kind() == reflect.Slice) {
		return
	}

	length := results.Len()
	if length > 0 {
		switch cursor.Order {
		case ASC:
			cursor.nextAfter = getCursorStringFromColumns(results.Index(length-1), columns...)
			cursor.nextBefore = getCursorStringFromColumns(results.Index(0), columns...)
		case DESC:
			cursor.nextAfter = getCursorStringFromColumns(results.Index(0), columns...)
			cursor.nextBefore = getCursorStringFromColumns(results.Index(length-1), columns...)
		}
	} else {
		if cursor.After != "" {
			cursor.nextAfter = cursor.After
		} else {
			cursor.nextAfter = getCursorStringFromColumns(reflect.New(results.Type().Elem()), columns...)
		}

		if cursor.Before != "" {
			cursor.nextBefore = cursor.Before
		} else {
			cursor.nextBefore = getCursorStringFromColumns(reflect.New(results.Type().Elem()), columns...)
		}
	}
}

func getCursorStringFromColumns(value reflect.Value, columns ...string) string {
	value = reflect.Indirect(value)
	if !(value.Kind() == reflect.Struct && value.Kind() == reflect.Struct) {
		panic("Find result is not a struct or an array of struct.")
	}
	if len(columns) == 0 {
		return ""
	}

	timeValue := value.FieldByName(columns[0])
	if timeValue == (reflect.Value{}) {
		panic(fmt.Sprintf("%s field does not exist", columns[0]))
	}

	t := new(time.Time)
	if !(timeValue.Type() == reflect.TypeOf(t) ||
		reflect.PtrTo(timeValue.Type()) == reflect.TypeOf(t)) {
		panic(fmt.Sprintf("%s field is not time.Time or *time.Time", columns[0]))
	}

	if !timeValue.CanInterface() {
		panic("timeValue can not interface")
	}

	if timeValue.Kind() == reflect.Ptr {
		t = timeValue.Interface().(*time.Time)
	} else {
		*t = timeValue.Interface().(time.Time)
	}

	args := make([]interface{}, len(columns)-1)
	for i, column := range columns[1:] {
		argValue := value.FieldByName(column)
		if argValue.CanInterface() {
			args[i] = argValue.Interface()
		} else {
			args[i] = nil
		}
	}

	return FormatCursorString(t, args...)
}
