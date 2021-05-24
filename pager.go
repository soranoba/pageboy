package pageboy

import (
	"math"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Pager is a builder that build a GORM scope that specifies a range of records.
type Pager struct {
	Page    int `json:"page"     query:"page"`
	PerPage int `json:"per_page" query:"per_page"`

	totalCount int64
}

// PagerSummary is summary of the query.
type PagerSummary struct {
	Page       int   `json:"page"        query:"page"`
	PerPage    int   `json:"per_page"    query:"per_page"`
	TotalCount int64 `json:"total_count" query:"total_count"`
	TotalPage  int   `json:"total_page"  query:"total_page"`
}

// NewPager returns a default Pager.
func NewPager() *Pager {
	return &Pager{
		Page:    1,
		PerPage: 10,
	}
}

// Summary returns a PagerSummary.
func (pager *Pager) Summary() *PagerSummary {
	return &PagerSummary{
		Page:       pager.Page,
		PerPage:    pager.PerPage,
		TotalCount: pager.totalCount,
		TotalPage:  int(math.Ceil(float64(pager.totalCount) / float64(pager.PerPage))),
	}
}

// Validate returns true when the values of Pager is valid. Otherwise, it returns false.
// If you execute Paginate with an invalid values, it panic may occur.
func (pager *Pager) Validate() error {
	if pager.PerPage == 0 {
		return &ValidationError{Field: "PerPage", Message: "must be greater than 0"}
	}
	if pager.Page == 0 {
		return &ValidationError{Field: "Page", Message: "must be greater than 0"}
	}
	return nil
}

// Scope returns a GORM scope.
func (pager *Pager) Scope() func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		db = db.InstanceSet("pageboy:pager", pager)
		return db.Offset((pager.Page - 1) * pager.PerPage).Limit(pager.PerPage)
	}
}

func pagerHandleBeforeQuery(db *gorm.DB) {
	value, ok := db.InstanceGet("pageboy:pager")
	if !ok {
		return
	}
	pager, ok := value.(*Pager)
	if !ok {
		return
	}
	if pager.totalCount > 0 {
		return
	}

	tx := db.Session(&gorm.Session{})
	clauses := tx.Statement.Clauses
	newClauses := make(map[string]clause.Clause)
	orderKey := (&clause.OrderBy{}).Name()
	limitKey := (&clause.Limit{}).Name()
	for k, v := range clauses {
		if k != orderKey && k != limitKey {
			newClauses[k] = v
		}
	}
	tx.Statement.Clauses = newClauses
	tx.Model(db.Statement.Dest).Count(&pager.totalCount)

	tx.Statement.Clauses = clauses
}

func registerPagerCallbacks(db *gorm.DB) {
	q := db.Callback().Query()
	q.Before("gorm:query").Replace("pageboy:pager:before_query", pagerHandleBeforeQuery)
}
