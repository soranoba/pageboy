package magion

import (
	"testing"
	"time"

	"github.com/jinzhu/gorm"
)

type pagerModel struct {
	gorm.Model
	name string
}

func TestPagerValidate(t *testing.T) {
	pager := Pager{Page: 0, PerPage: 1}
	assertError(t, pager.Validate())

	pager = Pager{Page: 1, PerPage: 0}
	assertError(t, pager.Validate())

	pager = Pager{Page: 1, PerPage: 1}
	assertNoError(t, pager.Validate())
}

func TestPagerPaginate(t *testing.T) {
	db := openDB()
	assertNoError(t, db.DropTableIfExists(&pagerModel{}).Error)
	assertNoError(t, db.AutoMigrate(&pagerModel{}).Error)

	now := time.Now()

	model1 := &pagerModel{
		Model: gorm.Model{
			CreatedAt: now,
		},
		name: "aaa",
	}
	assertNoError(t, db.Create(&model1).Error)

	model2 := &pagerModel{
		Model: gorm.Model{
			CreatedAt: now,
		},
		name: "bbb",
	}
	assertNoError(t, db.Create(&model2).Error)

	model3 := &pagerModel{
		Model: gorm.Model{
			CreatedAt: now.Add(10 * time.Millisecond),
		},
		name: "ccc",
	}
	assertNoError(t, db.Create(&model3).Error)

	model4 := &pagerModel{
		Model: gorm.Model{
			CreatedAt: now.Add(10 * time.Hour),
		},
		name: "ddd",
	}
	assertNoError(t, db.Create(&model4).Error)

	var models []*pagerModel
	pager := &Pager{Page: 1, PerPage: 2}
	assertNoError(t, db.Scopes(pager.Paginate()).Order("id ASC").Find(&models).Error)
	assertEqual(t, len(models), 2)
	assertEqual(t, models[0].ID, model1.ID)
	assertEqual(t, models[1].ID, model2.ID)
	assertEqual(t, *pager.Summary(), PagerSummary{Page: 1, PerPage: 2, TotalCount: 4, TotalPage: 2})

	pager = &Pager{Page: 2, PerPage: 3}
	assertNoError(t, db.Scopes(pager.Paginate()).Order("id ASC").Find(&models).Error)
	assertEqual(t, len(models), 1)
	assertEqual(t, models[0].ID, model4.ID)
	assertEqual(t, *pager.Summary(), PagerSummary{Page: 2, PerPage: 3, TotalCount: 4, TotalPage: 2})

	pager = &Pager{Page: 3, PerPage: 3}
	assertNoError(t, db.Scopes(pager.Paginate()).Order("id ASC").Find(&models).Error)
	assertEqual(t, len(models), 0)
	assertEqual(t, *pager.Summary(), PagerSummary{Page: 3, PerPage: 3, TotalCount: 4, TotalPage: 2})
}
