package pageboy

import (
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Cursor can to get a specific range of records from DB in time order.
//
// When Limit is smaller than or equal to 0 or Order is empty, the validation will fail.
// You should set the initial values and then read it from query or json.
//
//   cursor := &pageboy.Cursor{Limit: 10, Order: pageboy.DESC}
//   ctx.Bind(cursor)
//
type Cursor struct {
	Before string `json:"before" query:"before"`
	After  string `json:"after" query:"after"`
	Limit  int    `json:"limit" query:"limit"`
	Order  Order  `json:"order" query:"order" enums:"asc,desc"`

	nextBefore string
	nextAfter  string
	limit      int
	hasMore    bool
}

// CursorPagingUrls is for the user to access from the next cursor position.
// If it is no records at target of before or after, it will be empty strings.
type CursorPagingUrls struct {
	Before string `json:"before,omitempty"`
	After  string `json:"after,omitempty"`
}

// CursorSegment is a result of parsing each element of cursor
type CursorSegment struct {
	integer int64
	nano    int64
	isNil   bool
}

type CursorSegments []CursorSegment

// IsNil returns true if it have nil value. Otherwise, it returns false.
func (seg CursorSegment) IsNil() bool {
	return seg.isNil
}

// Int64 returns converted to integer.
func (seg CursorSegment) Int64() int64 {
	return seg.integer
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
	assert(ty.Kind() == reflect.Struct, "model must be struct")
	field, ok := ty.FieldByName(column)
	if !ok {
		return seg.Int64()
	}

	if field.Type == reflect.TypeOf(time.Time{}) ||
		field.Type == reflect.TypeOf(new(time.Time)) {
		return seg.Time()
	}
	return seg.Int64()
}

// Interface returns converted to types of specified columns.
func (segs CursorSegments) Interface(ty reflect.Type, columns ...string) []interface{} {
	assert(len(segs) == len(columns), "invalid number of columns")

	results := make([]interface{}, len(columns))
	for i, column := range columns {
		results[i] = segs[i].Interface(ty, column)
	}
	return results
}

// NewCursor returns a default Cursor.
func NewCursor() *Cursor {
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
//
// You can use GetNextBefore and GetNextAfter if you want to customize the behavior.
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
		// there are no newer elements within the specified range now.
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
func (cursor *Cursor) Paginate(columns ...string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		registerCursorCallbacks(db)
		db = db.InstanceSet("pageboy:columns", columns).
			InstanceSet("pageboy:cursor", cursor)

		return db.
			Order(CompositeOrder(cursor.Order, columns...)).
			Limit(cursor.Limit)
	}
}

// FormatCursorString returns a string for Cursor.
func FormatCursorString(args ...interface{}) string {
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

// ParseCursorString parses a string for cursor.
func ParseCursorString(str string) CursorSegments {
	parts := strings.Split(str, "_")

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

func getCursor(db *gorm.DB) (*Cursor, bool) {
	value, ok := db.InstanceGet("pageboy:cursor")
	if !ok {
		return nil, false
	}
	cursor, ok := value.(*Cursor)
	if !ok {
		return nil, false
	}
	return cursor, true
}

func getColumns(db *gorm.DB) ([]string, bool) {
	value, ok := db.InstanceGet("pageboy:columns")
	if !ok {
		return nil, false
	}
	columns, ok := value.([]string)
	if !ok {
		return nil, false
	}
	return columns, true
}

func cursorHandleBeforeQuery(db *gorm.DB) {
	cursor, ok := getCursor(db)
	if !ok {
		return
	}

	columns, ok := getColumns(db)
	if !ok {
		return
	}

	dest := db.Statement.Dest
	ty := reflect.TypeOf(dest)
	for ty.Kind() == reflect.Ptr || ty.Kind() == reflect.Array || ty.Kind() == reflect.Slice {
		ty = ty.Elem()
	}

	if cursor.Before != "" {
		segments := ParseCursorString(cursor.Before)
		args := segments.Interface(ty, columns...)
		db = db.Scopes(CompositeSortScopeFunc("<", columns...)(args...))
	}

	if cursor.After != "" {
		segments := ParseCursorString(cursor.After)
		args := segments.Interface(ty, columns...)
		db = db.Scopes(CompositeSortScopeFunc(">", columns...)(args...))
	}

	limit, ok := db.Statement.Clauses[new(clause.Limit).Name()]
	if ok {
		cursor.limit = limit.Expression.(clause.Limit).Limit
		db.Limit(cursor.limit + 1)
	} else {
		cursor.limit = -1
	}
}

func cursorHandleAfterQuery(db *gorm.DB) {
	cursor, ok := getCursor(db)
	if !ok {
		return
	}

	cursor.hasMore = false
	if cursor.limit == -1 {
		return
	}

	results := db.Statement.ReflectValue
	if !(results.Kind() == reflect.Array || results.Kind() == reflect.Slice) {
		return
	}

	if cursor.limit+1 == results.Len() {
		cursor.hasMore = true
		results.Set(results.Slice(0, results.Len()-1))
	}
}

func cursorHandleQuery(db *gorm.DB) {
	cursor, ok := getCursor(db)
	if !ok {
		return
	}
	columns, ok := getColumns(db)
	if !ok {
		return
	}

	cursor.nextBefore = ""
	cursor.nextAfter = ""

	if db.Error != nil {
		return
	}
	results := db.Statement.ReflectValue
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
		ty := results.Type().Elem()
		if ty.Kind() == reflect.Ptr {
			ty = ty.Elem()
		}

		if cursor.After != "" {
			cursor.nextAfter = cursor.After
		} else {
			cursor.nextAfter = getCursorStringFromColumns(reflect.New(ty), columns...)
		}

		if cursor.Before != "" {
			cursor.nextBefore = cursor.Before
		} else {
			cursor.nextBefore = getCursorStringFromColumns(reflect.New(ty), columns...)
		}
	}
}

func getCursorStringFromColumns(value reflect.Value, columns ...string) string {
	value = reflect.Indirect(value)
	if !(value.Kind() == reflect.Struct) {
		panic("Find result is not a struct or an array of struct.")
	}
	if len(columns) == 0 {
		return ""
	}

	args := make([]interface{}, len(columns))
	for i, column := range columns {
		argValue := value.FieldByName(column)
		if argValue.CanInterface() {
			args[i] = argValue.Interface()
		} else {
			args[i] = nil
		}
	}

	return FormatCursorString(args...)
}

func registerCursorCallbacks(db *gorm.DB) {
	q := db.Callback().Query()
	q.Before("gorm:query").Replace("pageboy:cursor:before_query", cursorHandleBeforeQuery)
	q.After("gorm:query").Replace("pageboy:cursor:after_query", cursorHandleAfterQuery)
	q.Replace("pageboy:cursor:handle_query", cursorHandleQuery)
}
