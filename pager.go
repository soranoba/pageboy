package magion

import (
	"errors"
	"math"

	"github.com/jinzhu/gorm"
)

// Pager can to get a specific range of records from DB.
type Pager struct {
	Page    uint `json:"page" query:"page"`
	PerPage uint `json:"per_page" query:"per_page"`

	totalCount uint
}

// PagerSummary is summary of the query.
type PagerSummary struct {
	Page       uint `json:"page" query:"page"`
	PerPage    uint `json:"per_page" query:"per_page"`
	TotalCount uint `json:"total_count" query:"total_count"`
	TotalPage  uint `json:"total_page" query:"total_page"`
}

func init() {
	gorm.DefaultCallback.Query().Before("gorm:query").
		Register("magion:pager:before_query", pagerHandleBeforeQuery)
}

// NewPager returns a default pager.
func NewPager() *Pager {
	return &Pager{
		Page:    1,
		PerPage: 10,
	}
}

// Summary returns summary of pager.
func (pager *Pager) Summary() *PagerSummary {
	return &PagerSummary{
		Page:       pager.Page,
		PerPage:    pager.PerPage,
		TotalCount: pager.totalCount,
		TotalPage:  uint(math.Ceil(float64(pager.totalCount) / float64(pager.PerPage))),
	}
}

// Validate returns true when the Pager is valid. Otherwise, it returns false.
// If you execute Paginate with an invalid value, panic may occur.
func (pager *Pager) Validate() error {
	if pager.PerPage == 0 {
		return errors.New("PerPage parameter must be greater than 0")
	}
	if pager.Page == 0 {
		return errors.New("Page parameter must be greater than 0")
	}
	return nil
}

// Paginate is a scope for the gorm.
//
// Example:
//
//   db.Scopes(pager.Paginate()).Order("id ASC").Find(&models)
//
func (pager *Pager) Paginate() func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		db = db.New().Set("magion:pager", pager)
		return db.Offset((pager.Page - 1) * pager.PerPage).Limit(pager.PerPage)
	}
}

func pagerHandleBeforeQuery(scope *gorm.Scope) {
	value, ok := scope.Get("magion:pager")
	if !ok {
		return
	}
	pager, ok := value.(*Pager)
	if !ok {
		return
	}
	scope.DB().NewScope(scope.DB().Value).DB().Offset(0).Limit(-1).Count(&pager.totalCount)
}
