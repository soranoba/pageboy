package pageboy

import (
	"math"

	"gorm.io/gorm"
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
		registerPagerCallbacks(db)
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

	tx := db.Session(&gorm.Session{WithConditions: true})
	tx.Offset(0).Limit(-1).
		Model(db.Statement.Dest).Count(&pager.totalCount)
}

func registerPagerCallbacks(db *gorm.DB) {
	q := db.Callback().Query()
	q.Before("gorm:query").Replace("pageboy:pager:before_query", pagerHandleBeforeQuery)
}
