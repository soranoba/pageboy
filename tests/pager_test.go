package pageboy_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/soranoba/pageboy"
	"gorm.io/gorm"
)

type pagerModel struct {
	gorm.Model
	Name string
}

func TestPagerValidate(t *testing.T) {
	pager := pageboy.Pager{Page: 0, PerPage: 1}
	assertError(t, pager.Validate())

	pager = pageboy.Pager{Page: 1, PerPage: 0}
	assertError(t, pager.Validate())

	pager = pageboy.Pager{Page: 1, PerPage: 1}
	assertNoError(t, pager.Validate())
}

func TestPagerPaginate(t *testing.T) {
	db := openDB()
	assertNoError(t, db.Migrator().DropTable(&pagerModel{}))
	assertNoError(t, db.AutoMigrate(&pagerModel{}))

	now := time.Now()

	model1 := &pagerModel{
		Model: gorm.Model{
			CreatedAt: now,
		},
	}
	assertNoError(t, db.Create(&model1).Error)

	model2 := &pagerModel{
		Model: gorm.Model{
			CreatedAt: now,
		},
	}
	assertNoError(t, db.Create(&model2).Error)

	model3 := &pagerModel{
		Model: gorm.Model{
			CreatedAt: now.Add(10 * time.Second),
		},
	}
	assertNoError(t, db.Create(&model3).Error)

	model4 := &pagerModel{
		Model: gorm.Model{
			CreatedAt: now.Add(10 * time.Hour),
		},
	}
	assertNoError(t, db.Create(&model4).Error)

	var models []*pagerModel
	pager := &pageboy.Pager{Page: 1, PerPage: 2}
	assertNoError(t, db.Scopes(pager.Scope()).Order("id ASC").Find(&models).Error)
	assertEqual(t, len(models), 2)
	assertEqual(t, models[0].ID, model1.ID)
	assertEqual(t, models[1].ID, model2.ID)
	assertEqual(t, *pager.Summary(), pageboy.PagerSummary{Page: 1, PerPage: 2, TotalCount: 4, TotalPage: 2})

	pager = &pageboy.Pager{Page: 2, PerPage: 3}
	assertNoError(t, db.Scopes(pager.Scope()).Order("id ASC").Find(&models).Error)
	assertEqual(t, len(models), 1)
	assertEqual(t, models[0].ID, model4.ID)
	assertEqual(t, *pager.Summary(), pageboy.PagerSummary{Page: 2, PerPage: 3, TotalCount: 4, TotalPage: 2})

	pager = &pageboy.Pager{Page: 3, PerPage: 3}
	assertNoError(t, db.Scopes(pager.Scope()).Order("id ASC").Find(&models).Error)
	assertEqual(t, len(models), 0)
	assertEqual(t, *pager.Summary(), pageboy.PagerSummary{Page: 3, PerPage: 3, TotalCount: 4, TotalPage: 2})
}

func TestPagerPaginateWithWhere(t *testing.T) {
	db := openDB().Debug()
	assertNoError(t, db.Migrator().DropTable(&pagerModel{}))
	assertNoError(t, db.AutoMigrate(&pagerModel{}))

	now := time.Now()

	model1 := &pagerModel{
		Model: gorm.Model{
			CreatedAt: now,
		},
		Name: "aaa",
	}
	assertNoError(t, db.Create(&model1).Error)

	model2 := &pagerModel{
		Model: gorm.Model{
			CreatedAt: now,
		},
		Name: "aaa",
	}
	assertNoError(t, db.Create(&model2).Error)

	model3 := &pagerModel{
		Model: gorm.Model{
			CreatedAt: now.Add(10 * time.Second),
		},
		Name: "ccc",
	}
	assertNoError(t, db.Create(&model3).Error)

	model4 := &pagerModel{
		Model: gorm.Model{
			CreatedAt: now.Add(10 * time.Hour),
		},
		Name: "ddd",
	}
	assertNoError(t, db.Create(&model4).Error)

	var models []*pagerModel
	pager := &pageboy.Pager{Page: 1, PerPage: 2}
	assertNoError(t, db.Scopes(pager.Scope()).Where("name = ?", "aaa").Order("id ASC").Find(&models).Error)
	assertEqual(t, len(models), 2)
	assertEqual(t, models[0].ID, model1.ID)
	assertEqual(t, models[1].ID, model2.ID)
	assertEqual(t, *pager.Summary(), pageboy.PagerSummary{Page: 1, PerPage: 2, TotalCount: 2, TotalPage: 1})
}

func ExamplePager() {
	db := openDB()

	type User struct {
		gorm.Model
		Name string
	}

	db.Migrator().DropTable(&User{})
	db.AutoMigrate(&User{})

	db.Create(&User{Name: "Alice"})
	db.Create(&User{Name: "Bob"})
	db.Create(&User{Name: "Carol"})

	// Default Values.
	pager := &pageboy.Pager{Page: 1, PerPage: 2}

	// Update values from a http request.

	// Fetch Records.
	var users []User
	db.Scopes(pager.Scope()).Order("id ASC").Find(&users)

	fmt.Printf("len(users) == %d\n", len(users))
	fmt.Printf("users[0].Name == \"%s\"\n", users[0].Name)
	fmt.Printf("users[1].Name == \"%s\"\n", users[1].Name)

	// Return the Summary.
	j, _ := json.Marshal(pager.Summary())
	fmt.Println(string(j))

	// Output:
	// len(users) == 2
	// users[0].Name == "Alice"
	// users[1].Name == "Bob"
	// {"page":1,"per_page":2,"total_count":3,"total_page":2}
}
