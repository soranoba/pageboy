package pageboy

import (
	"net/url"
	"reflect"
	"strings"

	"github.com/soranoba/pageboy/utils"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Cursor is a builder that build a GORM scope that specifies a range from the cursor position of records.
// You can read it from query or json.
type Cursor struct {
	Before  utils.CursorString `json:"before"  query:"before"`
	After   utils.CursorString `json:"after"   query:"after"`
	Limit   int                `json:"limit"   query:"limit"`
	Reverse bool               `json:"reverse" query:"reverse"`

	// See: cursor.Order
	orders []utils.Order
	// See: cursor.Paginate
	columns []string

	nextBefore utils.CursorString
	nextAfter  utils.CursorString
	baseOrder  utils.Order
	limit      int
	hasMore    bool
}

// CursorPagingUrls is for the user to access from the next cursor position.
// If it is no records at target of next, Next will be empty.
type CursorPagingUrls struct {
	Next string `json:"next,omitempty"`
}

// NewCursor returns a default Cursor.
func NewCursor() *Cursor {
	return &Cursor{
		Limit: 10,
	}
}

// Validate returns true when the Cursor is valid. Otherwise, it returns false.
// If you execute Paginate with an invalid value, it panic may occur.
func (cursor *Cursor) Validate() error {
	if cursor.Before != "" && !cursor.Before.Validate() {
		return &ValidationError{Field: "Before", Message: "is invalid"}
	}
	if cursor.After != "" && !cursor.After.Validate() {
		return &ValidationError{Field: "After", Message: "is invalid"}
	}
	if cursor.Limit < 1 {
		return &ValidationError{Field: "Limit", Message: "is invalid"}
	}
	return nil
}

// GetNextAfter returns a value of query to access if it exists some records after the current position.
func (cursor *Cursor) GetNextAfter() utils.CursorString {
	return cursor.nextAfter
}

// GetNextBefore returns a value of query to access if it exists some records before the current position.
func (cursor *Cursor) GetNextBefore() utils.CursorString {
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

	if cursor.hasMore {
		baseURL := *base
		query := baseURL.Query()
		if (cursor.baseOrder == utils.ASC) != cursor.Reverse {
			query.Del("after")
			query.Add("after", string(cursor.nextAfter))
		} else {
			query.Del("before")
			query.Add("before", string(cursor.nextBefore))
		}
		baseURL.RawQuery = query.Encode()
		pagingUrls.Next = baseURL.String()
	}

	return pagingUrls
}

// Paginate set the pagination target columns, and returns self.
func (cursor *Cursor) Paginate(columns ...string) *Cursor {
	cursor.columns = columns
	return cursor
}

// Order set the pagination orders, and returns self.
// The orders must be same order as columns that set to arguments of Paginate.
func (cursor *Cursor) Order(orders ...utils.Order) *Cursor {
	lowerOrders := make([]utils.Order, len(orders))
	for i := 0; i < len(orders); i++ {
		lowerOrders[i] = utils.Order(strings.ToLower(string(orders[i])))
	}
	cursor.orders = lowerOrders
	return cursor
}

// Scope returns a GORM scope.
func (cursor *Cursor) Scope() func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		registerCursorCallbacks(db)

		cursor.baseOrder = utils.ASC
		if len(cursor.orders) > 0 {
			cursor.baseOrder = cursor.orders[0]
		}

		db = db.InstanceSet("pageboy:cursor", cursor)

		if cursor.Reverse {
			db = db.Order(utils.OrderClauseBuilder(cursor.columns...)(utils.ReverseOrders(cursor.orders)...))
		} else {
			db = db.Order(utils.OrderClauseBuilder(cursor.columns...)(cursor.orders...))
		}
		return db.Limit(cursor.Limit)
	}
}

func (cursor *Cursor) comparisons(isBefore bool) []utils.Comparison {
	comparisons := make([]utils.Comparison, len(cursor.columns))
	ordersLength := len(cursor.orders)

	isReverse := func(order utils.Order) bool {
		if cursor.baseOrder == order {
			return false
		}
		return true
	}

	for i := 0; i < len(cursor.columns); i++ {
		order := func() utils.Order {
			if i < ordersLength {
				return cursor.orders[i]
			}
			return cursor.baseOrder
		}()

		if isBefore == isReverse(order) {
			comparisons[i] = utils.GreaterThan
		} else {
			comparisons[i] = utils.LessThan
		}
	}
	return comparisons
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

func cursorHandleBeforeQuery(db *gorm.DB) {
	cursor, ok := getCursor(db)
	if !ok {
		return
	}

	dest := db.Statement.Dest
	ty := reflect.TypeOf(dest)
	for ty.Kind() == reflect.Ptr || ty.Kind() == reflect.Array || ty.Kind() == reflect.Slice {
		ty = ty.Elem()
	}

	if cursor.Before != "" {
		segments := utils.NewCursorSegments(cursor.Before)
		args := segments.Interface(ty, cursor.columns...)
		db = db.Scopes(utils.MakeComparisonScopeBuildFunc(cursor.columns...)(cursor.comparisons(true)...)(args...))
	}

	if cursor.After != "" {
		segments := utils.NewCursorSegments(cursor.After)
		args := segments.Interface(ty, cursor.columns...)
		db = db.Scopes(utils.MakeComparisonScopeBuildFunc(cursor.columns...)(cursor.comparisons(false)...)(args...))
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
		if (cursor.baseOrder == utils.ASC) != cursor.Reverse {
			cursor.nextAfter = getCursorStringFromColumns(results.Index(length-1), cursor.columns...)
			cursor.nextBefore = getCursorStringFromColumns(results.Index(0), cursor.columns...)
		} else {
			cursor.nextAfter = getCursorStringFromColumns(results.Index(0), cursor.columns...)
			cursor.nextBefore = getCursorStringFromColumns(results.Index(length-1), cursor.columns...)
		}
	} else {
		ty := results.Type().Elem()
		if ty.Kind() == reflect.Ptr {
			ty = ty.Elem()
		}

		if cursor.After != "" {
			cursor.nextAfter = cursor.After
		} else {
			cursor.nextAfter = getCursorStringFromColumns(reflect.New(ty), cursor.columns...)
		}

		if cursor.Before != "" {
			cursor.nextBefore = cursor.Before
		} else {
			cursor.nextBefore = getCursorStringFromColumns(reflect.New(ty), cursor.columns...)
		}
	}
}

func getCursorStringFromColumns(value reflect.Value, columns ...string) utils.CursorString {
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

	return utils.FormatCursorString(args...)
}

func registerCursorCallbacks(db *gorm.DB) {
	q := db.Callback().Query()
	q.Before("gorm:query").Replace("pageboy:cursor:before_query", cursorHandleBeforeQuery)
	q.After("gorm:query").Replace("pageboy:cursor:after_query", cursorHandleAfterQuery)
	q.Replace("pageboy:cursor:handle_query", cursorHandleQuery)
}
