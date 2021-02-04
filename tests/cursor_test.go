package pageboy_test

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/soranoba/pageboy/v2"
	pbc "github.com/soranoba/pageboy/v2/core"
	"gorm.io/gorm"
)

type cursorModel struct {
	gorm.Model
	SubID *uint
	Name  string
	Time  *time.Time
}

type childModel struct {
	gorm.Model
}

var (
	DESC string = "DESC"
	ASC  string = "ASC"
)

func buildURL(cursor *pageboy.Cursor, base url.URL) *url.URL {
	query := base.Query()
	if cursor.Before != "" {
		query.Del("before")
		query.Add("before", string(cursor.Before))
	}
	if cursor.After != "" {
		query.Del("after")
		query.Add("after", string(cursor.After))
	}
	if cursor.Limit != 0 {
		query.Del("limit")
		query.Add("limit", strconv.Itoa(cursor.Limit))
	}
	if cursor.Reverse {
		query.Del("reverse")
		query.Add("reverse", "true")
	}

	base.RawQuery = query.Encode()
	return &base
}

func TestCursorValidate(t *testing.T) {
	// invalid before params
	cursor := &pageboy.Cursor{Before: "aaa", After: "", Limit: 10}
	assertError(t, cursor.Validate())

	// invalid after params
	cursor = &pageboy.Cursor{Before: "", After: "aaa", Limit: 10}
	assertError(t, cursor.Validate())

	// invalid limit params
	cursor = &pageboy.Cursor{Before: "1585706584", After: "1585706584"}
	assertError(t, cursor.Validate())
	cursor = &pageboy.Cursor{Before: "1585706584", After: "1585706584", Limit: -1}
	assertError(t, cursor.Validate())

	cursor = &pageboy.Cursor{Before: "1585706584.025_20", After: "", Limit: 10}
	assertNoError(t, cursor.Validate())

	cursor = &pageboy.Cursor{Before: "", After: "1585706584.025_20", Limit: 10}
	assertNoError(t, cursor.Validate())

	cursor = &pageboy.Cursor{Before: "1585706584", After: "1585706584", Limit: 10}
	assertNoError(t, cursor.Validate())

	cursor = &pageboy.Cursor{Before: "", After: "", Limit: 10}
	assertNoError(t, cursor.Validate())

	cursor = &pageboy.Cursor{Before: "", After: "", Limit: 10}
	assertNoError(t, cursor.Validate())
}

func TestCursorPaginateDESC(t *testing.T) {
	db := openDB()
	assertNoError(t, db.Migrator().DropTable(&cursorModel{}))
	assertNoError(t, db.AutoMigrate(&cursorModel{}))

	baseURL, err := url.Parse("https://example.com/users?a=1")
	assertNoError(t, err)

	now := time.Now()

	create := func(createdAt time.Time) *cursorModel {
		model := &cursorModel{
			Model: gorm.Model{
				CreatedAt: createdAt,
			},
		}
		assertNoError(t, db.Create(model).Error)
		return model
	}

	model1 := create(now)
	model2 := create(now)
	model3 := create(now.Add(10 * time.Second))
	model4 := create(now.Add(10 * time.Hour))

	var models []*cursorModel
	cursor := &pageboy.Cursor{
		Limit: 1,
	}
	url := buildURL(cursor, *baseURL)
	assertNoError(t, db.Scopes(cursor.Paginate("CreatedAt", "ID").Order(DESC, DESC).Scope()).Find(&models).Error)
	assertEqual(t, len(models), 1)
	assertEqual(t, models[0].ID, model4.ID)
	assertEqual(t, cursor.GetNextAfter(), pbc.FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, cursor.GetNextBefore(), pbc.FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, *cursor.BuildNextPagingUrls(url), pageboy.CursorPagingUrls{
		Next: string("https://example.com/users?a=1" +
			"&before=" + pbc.FormatCursorString(&models[0].CreatedAt, models[0].ID) +
			"&limit=1"),
	})

	cursor = &pageboy.Cursor{
		Before: cursor.GetNextBefore(),
		Limit:  2,
	}
	url = buildURL(cursor, *baseURL)
	assertNoError(t, db.Scopes(cursor.Paginate("CreatedAt", "ID").Order(DESC, DESC).Scope()).Find(&models).Error)
	assertEqual(t, len(models), 2)
	assertEqual(t, models[0].ID, model3.ID)
	assertEqual(t, models[1].ID, model2.ID)
	assertEqual(t, cursor.GetNextAfter(), pbc.FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, cursor.GetNextBefore(), pbc.FormatCursorString(&models[1].CreatedAt, models[1].ID))
	assertEqual(t, *cursor.BuildNextPagingUrls(url), pageboy.CursorPagingUrls{
		Next: "https://example.com/users?a=1" +
			"&before=" + string(pbc.FormatCursorString(&models[1].CreatedAt, models[1].ID)) +
			"&limit=2",
	})

	cursor = &pageboy.Cursor{
		Before: cursor.GetNextBefore(),
		Limit:  2,
	}
	url = buildURL(cursor, *baseURL)
	assertNoError(t, db.Scopes(cursor.Paginate("CreatedAt", "ID").Order(DESC, DESC).Scope()).Find(&models).Error)
	assertEqual(t, len(models), 1)
	assertEqual(t, models[0].ID, model1.ID)
	assertEqual(t, cursor.GetNextAfter(), pbc.FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, cursor.GetNextBefore(), pbc.FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, *cursor.BuildNextPagingUrls(url), pageboy.CursorPagingUrls{
		Next: "",
	})
}

func TestCursorPaginateASC(t *testing.T) {
	db := openDB()
	assertNoError(t, db.Migrator().DropTable(&cursorModel{}))
	assertNoError(t, db.AutoMigrate(&cursorModel{}))

	baseURL, err := url.Parse("https://example.com/users?a=1")
	assertNoError(t, err)

	now := time.Now()

	create := func(createdAt time.Time) *cursorModel {
		model := &cursorModel{
			Model: gorm.Model{
				CreatedAt: createdAt,
			},
		}
		assertNoError(t, db.Create(model).Error)
		return model
	}

	model1 := create(now)
	model2 := create(now)
	model3 := create(now.Add(10 * time.Second))
	model4 := create(now.Add(10 * time.Hour))

	var models []*cursorModel
	cursor := &pageboy.Cursor{
		Limit: 1,
	}
	url := buildURL(cursor, *baseURL)
	assertNoError(t, db.Scopes(cursor.Paginate("CreatedAt", "ID").Order(ASC, ASC).Scope()).Find(&models).Error)
	assertEqual(t, len(models), 1)
	assertEqual(t, models[0].ID, model1.ID)
	assertEqual(t, cursor.GetNextAfter(), pbc.FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, cursor.GetNextBefore(), pbc.FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, *cursor.BuildNextPagingUrls(url), pageboy.CursorPagingUrls{
		Next: "https://example.com/users?a=1" +
			"&after=" + string(pbc.FormatCursorString(&models[0].CreatedAt, models[0].ID)) +
			"&limit=1",
	})

	cursor = &pageboy.Cursor{
		After: cursor.GetNextAfter(),
		Limit: 2,
	}
	url = buildURL(cursor, *baseURL)
	assertNoError(t, db.Scopes(cursor.Paginate("CreatedAt", "ID").Order(ASC, ASC).Scope()).Find(&models).Error)
	assertEqual(t, len(models), 2)
	assertEqual(t, models[0].ID, model2.ID)
	assertEqual(t, models[1].ID, model3.ID)
	assertEqual(t, cursor.GetNextAfter(), pbc.FormatCursorString(&models[1].CreatedAt, models[1].ID))
	assertEqual(t, cursor.GetNextBefore(), pbc.FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, *cursor.BuildNextPagingUrls(url), pageboy.CursorPagingUrls{
		Next: "https://example.com/users?a=1" +
			"&after=" + string(pbc.FormatCursorString(&models[1].CreatedAt, models[1].ID)) +
			"&limit=2",
	})

	cursor = &pageboy.Cursor{
		After: cursor.GetNextAfter(),
		Limit: 2,
	}
	url = buildURL(cursor, *baseURL)
	assertNoError(t, db.Scopes(cursor.Paginate("CreatedAt", "ID").Order(ASC, ASC).Scope()).Find(&models).Error)
	assertEqual(t, len(models), 1)
	assertEqual(t, models[0].ID, model4.ID)
	assertEqual(t, cursor.GetNextAfter(), pbc.FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, cursor.GetNextBefore(), pbc.FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, *cursor.BuildNextPagingUrls(url), pageboy.CursorPagingUrls{
		Next: "",
	})
}

func TestCursorPaginateWithBeforeDESC(t *testing.T) {
	db := openDB()
	assertNoError(t, db.Migrator().DropTable(&cursorModel{}))
	assertNoError(t, db.AutoMigrate(&cursorModel{}))

	baseURL, err := url.Parse("https://example.com/users?a=1")
	assertNoError(t, err)

	now := time.Now()
	now = now.Add(-1 * time.Duration(now.Nanosecond()) * time.Nanosecond)

	create := func(createdAt time.Time) *cursorModel {
		model := &cursorModel{
			Model: gorm.Model{
				CreatedAt: createdAt,
			},
		}
		assertNoError(t, db.Create(model).Error)
		return model
	}

	model1 := create(now)
	model2 := create(now)
	model3 := create(now.Add(10 * time.Second))
	model4 := create(now.Add(10 * time.Hour))

	var models []*cursorModel
	cursor := &pageboy.Cursor{
		Before: pbc.FormatCursorString(&model4.CreatedAt, model4.ID),
		Limit:  2,
	}
	url := buildURL(cursor, *baseURL)
	assertNoError(t, db.Scopes(cursor.Paginate("CreatedAt", "ID").Order(DESC, DESC).Scope()).Find(&models).Error)
	assertEqual(t, len(models), 2)
	assertEqual(t, models[0].ID, model3.ID)
	assertEqual(t, models[1].ID, model2.ID)
	assertEqual(t, cursor.GetNextAfter(), pbc.FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, cursor.GetNextBefore(), pbc.FormatCursorString(&models[1].CreatedAt, models[1].ID))
	assertEqual(t, *cursor.BuildNextPagingUrls(url), pageboy.CursorPagingUrls{
		Next: "https://example.com/users?a=1" +
			"&before=" + string(pbc.FormatCursorString(&models[1].CreatedAt, models[1].ID)) +
			"&limit=2",
	})

	cursor = &pageboy.Cursor{
		Before: cursor.GetNextBefore(),
		Limit:  2,
	}
	url = buildURL(cursor, *baseURL)
	assertNoError(t, db.Scopes(cursor.Paginate("CreatedAt", "ID").Order(DESC, DESC).Scope()).Find(&models).Error)
	assertEqual(t, len(models), 1)
	assertEqual(t, models[0].ID, model1.ID)
	assertEqual(t, cursor.GetNextAfter(), pbc.FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, cursor.GetNextBefore(), pbc.FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, *cursor.BuildNextPagingUrls(url), pageboy.CursorPagingUrls{
		Next: "",
	})
}

func TestCursorPaginateWithBeforeASC(t *testing.T) {
	db := openDB()
	assertNoError(t, db.Migrator().DropTable(&cursorModel{}))
	assertNoError(t, db.AutoMigrate(&cursorModel{}))

	baseURL, err := url.Parse("https://example.com/users?a=1")
	assertNoError(t, err)

	now := time.Now()
	now = now.Add(-1 * time.Duration(now.Nanosecond()) * time.Nanosecond)

	create := func(createdAt time.Time) *cursorModel {
		model := &cursorModel{
			Model: gorm.Model{
				CreatedAt: createdAt,
			},
		}
		assertNoError(t, db.Create(model).Error)
		return model
	}

	model1 := create(now)
	model2 := create(now)
	model3 := create(now.Add(10 * time.Second))
	model4 := create(now.Add(10 * time.Hour))

	var models []*cursorModel
	cursor := &pageboy.Cursor{
		Before: pbc.FormatCursorString(&model4.CreatedAt, model4.ID),
		Limit:  2,
	}
	url := buildURL(cursor, *baseURL)
	assertNoError(t, db.Scopes(cursor.Paginate("CreatedAt", "ID").Order(ASC, ASC).Scope()).Find(&models).Error)
	assertEqual(t, len(models), 2)
	assertEqual(t, models[0].ID, model1.ID)
	assertEqual(t, models[1].ID, model2.ID)
	assertEqual(t, cursor.GetNextAfter(), pbc.FormatCursorString(&models[1].CreatedAt, models[1].ID))
	assertEqual(t, cursor.GetNextBefore(), pbc.FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, *cursor.BuildNextPagingUrls(url), pageboy.CursorPagingUrls{
		Next: "https://example.com/users?a=1" +
			"&after=" + string(pbc.FormatCursorString(&models[1].CreatedAt, models[1].ID)) +
			"&before=" + string(cursor.Before) +
			"&limit=2",
	})

	cursor = &pageboy.Cursor{
		After:  cursor.GetNextAfter(),
		Before: cursor.Before,
		Limit:  2,
	}
	url = buildURL(cursor, *baseURL)
	assertNoError(t, db.Scopes(cursor.Paginate("CreatedAt", "ID").Order(ASC, ASC).Scope()).Find(&models).Error)
	assertEqual(t, len(models), 1)
	assertEqual(t, models[0].ID, model3.ID)
	assertEqual(t, cursor.GetNextAfter(), pbc.FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, cursor.GetNextBefore(), pbc.FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, *cursor.BuildNextPagingUrls(url), pageboy.CursorPagingUrls{
		Next: "",
	})
}

func TestCursorPaginateWithAfterDESC(t *testing.T) {
	db := openDB()
	assertNoError(t, db.Migrator().DropTable(&cursorModel{}))
	assertNoError(t, db.AutoMigrate(&cursorModel{}))

	baseURL, err := url.Parse("https://example.com/users?a=1")
	assertNoError(t, err)

	now := time.Now()
	now = now.Add(-1 * time.Duration(now.Nanosecond()) * time.Nanosecond)

	create := func(createdAt time.Time) *cursorModel {
		model := &cursorModel{
			Model: gorm.Model{
				CreatedAt: createdAt,
			},
		}
		assertNoError(t, db.Create(model).Error)
		return model
	}

	model1 := create(now)
	model2 := create(now.Add(10 * time.Second))
	model3 := create(now.Add(10 * time.Second))
	model4 := create(now.Add(10 * time.Hour))

	var models []*cursorModel
	cursor := &pageboy.Cursor{
		After: pbc.FormatCursorString(&model1.CreatedAt, model1.ID),
		Limit: 2,
	}
	url := buildURL(cursor, *baseURL)
	assertNoError(t, db.Scopes(cursor.Paginate("CreatedAt", "ID").Order(DESC, DESC).Scope()).Find(&models).Error)
	assertEqual(t, len(models), 2)
	assertEqual(t, models[0].ID, model4.ID)
	assertEqual(t, models[1].ID, model3.ID)
	assertEqual(t, cursor.GetNextAfter(), pbc.FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, cursor.GetNextBefore(), pbc.FormatCursorString(&models[1].CreatedAt, models[1].ID))
	assertEqual(t, *cursor.BuildNextPagingUrls(url), pageboy.CursorPagingUrls{
		Next: "https://example.com/users?a=1" +
			"&after=" + string(cursor.After) +
			"&before=" + string(pbc.FormatCursorString(&models[1].CreatedAt, models[1].ID)) +
			"&limit=2",
	})

	cursor = &pageboy.Cursor{
		After:  cursor.After,
		Before: cursor.GetNextBefore(),
		Limit:  2,
	}
	url = buildURL(cursor, *baseURL)
	assertNoError(t, db.Scopes(cursor.Paginate("CreatedAt", "ID").Order(DESC, DESC).Scope()).Find(&models).Error)
	assertEqual(t, len(models), 1)
	assertEqual(t, models[0].ID, model2.ID)
	assertEqual(t, cursor.GetNextAfter(), pbc.FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, cursor.GetNextBefore(), pbc.FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, *cursor.BuildNextPagingUrls(url), pageboy.CursorPagingUrls{
		Next: "",
	})
}

func TestCursorPaginateWithAfterASC(t *testing.T) {
	db := openDB()
	assertNoError(t, db.Migrator().DropTable(&cursorModel{}))
	assertNoError(t, db.AutoMigrate(&cursorModel{}))

	baseURL, err := url.Parse("https://example.com/users?a=1")
	assertNoError(t, err)

	now := time.Now()
	now = now.Add(-1 * time.Duration(now.Nanosecond()) * time.Nanosecond)

	create := func(createdAt time.Time) *cursorModel {
		model := &cursorModel{
			Model: gorm.Model{
				CreatedAt: createdAt,
			},
		}
		assertNoError(t, db.Create(model).Error)
		return model
	}

	model1 := create(now)
	model2 := create(now.Add(10 * time.Second))
	model3 := create(now.Add(10 * time.Second))
	model4 := create(now.Add(10 * time.Hour))

	var models []*cursorModel
	cursor := &pageboy.Cursor{
		After: pbc.FormatCursorString(&model1.CreatedAt, model1.ID),
		Limit: 2,
	}
	url := buildURL(cursor, *baseURL)
	assertNoError(t, db.Scopes(cursor.Paginate("CreatedAt", "ID").Order(ASC, ASC).Scope()).Find(&models).Error)
	assertEqual(t, len(models), 2)
	assertEqual(t, models[0].ID, model2.ID)
	assertEqual(t, models[1].ID, model3.ID)
	assertEqual(t, cursor.GetNextAfter(), pbc.FormatCursorString(&models[1].CreatedAt, models[1].ID))
	assertEqual(t, cursor.GetNextBefore(), pbc.FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, *cursor.BuildNextPagingUrls(url), pageboy.CursorPagingUrls{
		Next: "https://example.com/users?a=1" +
			"&after=" + string(pbc.FormatCursorString(&models[1].CreatedAt, models[1].ID)) +
			"&limit=2",
	})

	cursor = &pageboy.Cursor{
		After: cursor.GetNextAfter(),
		Limit: 2,
	}
	url = buildURL(cursor, *baseURL)
	assertNoError(t, db.Scopes(cursor.Paginate("CreatedAt", "ID").Order(ASC, ASC).Scope()).Find(&models).Error)
	assertEqual(t, len(models), 1)
	assertEqual(t, models[0].ID, model4.ID)
	assertEqual(t, cursor.GetNextAfter(), pbc.FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, cursor.GetNextBefore(), pbc.FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, *cursor.BuildNextPagingUrls(url), pageboy.CursorPagingUrls{
		Next: "",
	})
}

func TestCursorPaginateWithAfterAndBeforeDESC(t *testing.T) {
	db := openDB()
	assertNoError(t, db.Migrator().DropTable(&cursorModel{}))
	assertNoError(t, db.AutoMigrate(&cursorModel{}))

	baseURL, err := url.Parse("https://example.com/users?a=1")
	assertNoError(t, err)

	now := time.Now()
	now = now.Add(-1 * time.Duration(now.Nanosecond()) * time.Nanosecond)

	create := func(createdAt time.Time) *cursorModel {
		model := &cursorModel{
			Model: gorm.Model{
				CreatedAt: createdAt,
			},
		}
		assertNoError(t, db.Create(model).Error)
		return model
	}

	model1 := create(now)
	model2 := create(now.Add(10 * time.Second))
	model3 := create(now.Add(10 * time.Second))
	model4 := create(now.Add(10 * time.Hour))
	model5 := create(now.Add(10 * time.Hour))

	var models []*cursorModel
	cursor := &pageboy.Cursor{
		Before: pbc.FormatCursorString(&model5.CreatedAt, model5.ID),
		After:  pbc.FormatCursorString(&model1.CreatedAt, model1.ID),
		Limit:  2,
	}
	url := buildURL(cursor, *baseURL)
	assertNoError(t, db.Scopes(cursor.Paginate("CreatedAt", "ID").Order(DESC, DESC).Scope()).Find(&models).Error)
	assertEqual(t, len(models), 2)
	assertEqual(t, models[0].ID, model4.ID)
	assertEqual(t, models[1].ID, model3.ID)
	assertEqual(t, cursor.GetNextAfter(), pbc.FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, cursor.GetNextBefore(), pbc.FormatCursorString(&models[1].CreatedAt, models[1].ID))
	assertEqual(t, *cursor.BuildNextPagingUrls(url), pageboy.CursorPagingUrls{
		Next: "https://example.com/users?a=1" +
			"&after=" + string(cursor.After) +
			"&before=" + string(pbc.FormatCursorString(&models[1].CreatedAt, models[1].ID)) +
			"&limit=2",
	})

	cursor = &pageboy.Cursor{
		Before: cursor.GetNextBefore(),
		After:  cursor.After,
		Limit:  2,
	}
	url = buildURL(cursor, *baseURL)
	assertNoError(t, db.Scopes(cursor.Paginate("CreatedAt", "ID").Order(DESC, DESC).Scope()).Find(&models).Error)
	assertEqual(t, len(models), 1)
	assertEqual(t, models[0].ID, model2.ID)
	assertEqual(t, cursor.GetNextAfter(), pbc.FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, cursor.GetNextBefore(), pbc.FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, *cursor.BuildNextPagingUrls(url), pageboy.CursorPagingUrls{
		Next: "",
	})
}

func TestCursorPaginateWithAfterAndBeforeASC(t *testing.T) {
	db := openDB()
	assertNoError(t, db.Migrator().DropTable(&cursorModel{}))
	assertNoError(t, db.AutoMigrate(&cursorModel{}))

	baseURL, err := url.Parse("https://example.com/users?a=1")
	assertNoError(t, err)

	now := time.Now()
	now = now.Add(-1 * time.Duration(now.Nanosecond()) * time.Nanosecond)

	create := func(createdAt time.Time) *cursorModel {
		model := &cursorModel{
			Model: gorm.Model{
				CreatedAt: createdAt,
			},
		}
		assertNoError(t, db.Create(model).Error)
		return model
	}

	model1 := create(now)
	model2 := create(now.Add(10 * time.Second))
	model3 := create(now.Add(10 * time.Second))
	model4 := create(now.Add(10 * time.Hour))
	model5 := create(now.Add(10 * time.Hour))

	var models []*cursorModel
	cursor := &pageboy.Cursor{
		Before: pbc.FormatCursorString(&model5.CreatedAt, model5.ID),
		After:  pbc.FormatCursorString(&model1.CreatedAt, model1.ID),
		Limit:  2,
	}
	url := buildURL(cursor, *baseURL)
	assertNoError(t, db.Scopes(cursor.Paginate("CreatedAt", "ID").Order(ASC, ASC).Scope()).Find(&models).Error)
	assertEqual(t, len(models), 2)
	assertEqual(t, models[0].ID, model2.ID)
	assertEqual(t, models[1].ID, model3.ID)
	assertEqual(t, cursor.GetNextAfter(), pbc.FormatCursorString(&models[1].CreatedAt, models[1].ID))
	assertEqual(t, cursor.GetNextBefore(), pbc.FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, *cursor.BuildNextPagingUrls(url), pageboy.CursorPagingUrls{
		Next: "https://example.com/users?a=1" +
			"&after=" + string(pbc.FormatCursorString(&models[1].CreatedAt, models[1].ID)) +
			"&before=" + string(cursor.Before) +
			"&limit=2",
	})

	cursor = &pageboy.Cursor{
		Before: cursor.Before,
		After:  cursor.GetNextAfter(),
		Limit:  2,
	}
	url = buildURL(cursor, *baseURL)
	assertNoError(t, db.Scopes(cursor.Paginate("CreatedAt", "ID").Order(ASC, ASC).Scope()).Find(&models).Error)
	assertEqual(t, len(models), 1)
	assertEqual(t, models[0].ID, model4.ID)
	assertEqual(t, cursor.GetNextAfter(), pbc.FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, cursor.GetNextBefore(), pbc.FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, *cursor.BuildNextPagingUrls(url), pageboy.CursorPagingUrls{
		Next: "",
	})
}

func TestCursorPaginateWithEmpty(t *testing.T) {
	db := openDB()
	assertNoError(t, db.Migrator().DropTable(&cursorModel{}))
	assertNoError(t, db.AutoMigrate(&cursorModel{}))

	var models []*cursorModel
	cursor := &pageboy.Cursor{
		Limit: 1,
	}
	assertNoError(t, db.Scopes(cursor.Paginate("Time", "ID").Order(DESC, DESC).Scope()).Find(&models).Error)
	assertEqual(t, len(models), 0)
	assertEqual(t, cursor.GetNextAfter(), pbc.CursorString("_0"))
	assertEqual(t, cursor.GetNextBefore(), pbc.CursorString("_0"))
}

func TestCursorPaginateWithNullableTimeDESC(t *testing.T) {
	db := openDB()
	assertNoError(t, db.Migrator().DropTable(&cursorModel{}))
	assertNoError(t, db.AutoMigrate(&cursorModel{}))

	now := time.Now()

	create := func(ti *time.Time) *cursorModel {
		model := &cursorModel{
			Time: ti,
		}
		assertNoError(t, db.Create(model).Error)
		return model
	}

	model1 := create(nil)
	model2 := create(nil)
	model3 := create(&now)
	model4Time := now.Add(10 * time.Second)
	model4 := create(&model4Time)

	var models []*cursorModel
	cursor := &pageboy.Cursor{
		Limit: 1,
	}
	assertNoError(t, db.Scopes(cursor.Paginate("Time", "ID").Order(DESC, DESC).Scope()).Find(&models).Error)
	assertEqual(t, len(models), 1)
	assertEqual(t, models[0].ID, model4.ID)
	assertEqual(t, cursor.GetNextAfter(), pbc.FormatCursorString(models[0].Time, models[0].ID))
	assertEqual(t, cursor.GetNextBefore(), pbc.FormatCursorString(models[0].Time, models[0].ID))

	cursor = &pageboy.Cursor{
		Before: cursor.GetNextBefore(),
		Limit:  2,
	}
	assertNoError(t, db.Scopes(cursor.Paginate("Time", "ID").Order(DESC, DESC).Scope()).Find(&models).Error)
	assertEqual(t, len(models), 2)
	assertEqual(t, models[0].ID, model3.ID)
	assertEqual(t, models[1].ID, model2.ID)
	assertEqual(t, cursor.GetNextAfter(), pbc.FormatCursorString(models[0].Time, models[0].ID))
	assertEqual(t, cursor.GetNextBefore(), pbc.FormatCursorString(models[1].Time, models[1].ID))

	cursor = &pageboy.Cursor{
		Before: cursor.GetNextBefore(),
		Limit:  2,
	}
	assertNoError(t, db.Scopes(cursor.Paginate("Time", "ID").Order(DESC, DESC).Scope()).Find(&models).Error)
	assertEqual(t, len(models), 1)
	assertEqual(t, models[0].ID, model1.ID)
	assertEqual(t, cursor.GetNextAfter(), pbc.FormatCursorString(models[0].Time, models[0].ID))
	assertEqual(t, cursor.GetNextBefore(), pbc.FormatCursorString(models[0].Time, models[0].ID))
}

func TestCursorPaginateWithNullableTimeASC(t *testing.T) {
	db := openDB()
	assertNoError(t, db.Migrator().DropTable(&cursorModel{}))
	assertNoError(t, db.AutoMigrate(&cursorModel{}))

	now := time.Now()

	create := func(ti *time.Time) *cursorModel {
		model := &cursorModel{
			Time: ti,
		}
		assertNoError(t, db.Create(model).Error)
		return model
	}

	model1 := create(nil)
	model2 := create(nil)
	model3 := create(&now)
	model4Time := now.Add(10 * time.Second)
	model4 := create(&model4Time)

	var models []*cursorModel
	cursor := &pageboy.Cursor{
		Limit: 1,
	}
	assertNoError(t, db.Scopes(cursor.Paginate("Time", "ID").Order(ASC, ASC).Scope()).Find(&models).Error)
	assertEqual(t, len(models), 1)
	assertEqual(t, models[0].ID, model1.ID)
	assertEqual(t, cursor.GetNextBefore(), pbc.FormatCursorString(models[0].Time, models[0].ID))
	assertEqual(t, cursor.GetNextAfter(), pbc.FormatCursorString(models[0].Time, models[0].ID))

	cursor = &pageboy.Cursor{
		After: cursor.GetNextAfter(),
		Limit: 2,
	}
	assertNoError(t, db.Scopes(cursor.Paginate("Time", "ID").Order(ASC, ASC).Scope()).Find(&models).Error)
	assertEqual(t, len(models), 2)
	assertEqual(t, models[0].ID, model2.ID)
	assertEqual(t, models[1].ID, model3.ID)
	assertEqual(t, cursor.GetNextBefore(), pbc.FormatCursorString(models[0].Time, models[0].ID))
	assertEqual(t, cursor.GetNextAfter(), pbc.FormatCursorString(models[1].Time, models[1].ID))

	cursor = &pageboy.Cursor{
		After: cursor.GetNextAfter(),
		Limit: 2,
	}
	assertNoError(t, db.Scopes(cursor.Paginate("Time", "ID").Order(ASC, ASC).Scope()).Find(&models).Error)
	assertEqual(t, len(models), 1)
	assertEqual(t, models[0].ID, model4.ID)
	assertEqual(t, cursor.GetNextBefore(), pbc.FormatCursorString(models[0].Time, models[0].ID))
	assertEqual(t, cursor.GetNextAfter(), pbc.FormatCursorString(models[0].Time, models[0].ID))
}

func TestCursorPaginateWithNullableDescAsc(t *testing.T) {
	db := openDB()
	assertNoError(t, db.Migrator().DropTable(&cursorModel{}))
	assertNoError(t, db.AutoMigrate(&cursorModel{}))

	baseURL, err := url.Parse("https://example.com/users?a=1")
	assertNoError(t, err)

	create := func(subid uint) *cursorModel {
		var s *uint
		if subid != 0 {
			s = &subid
		}
		model := &cursorModel{
			SubID: s,
		}
		assertNoError(t, db.Create(model).Error)
		return model
	}

	model1 := create(3)
	model2 := create(0)
	model3 := create(2)
	model4 := create(0)
	model5 := create(0)

	var models []*cursorModel
	cursor := &pageboy.Cursor{
		Limit: 1,
	}
	url := buildURL(cursor, *baseURL)
	assertNoError(t, db.Scopes(cursor.Paginate("SubID", "ID").Order(DESC, ASC).Scope()).Find(&models).Error)
	assertEqual(t, len(models), 1)
	assertEqual(t, models[0].ID, model1.ID)
	assertEqual(t, cursor.GetNextAfter(), pbc.FormatCursorString(models[0].SubID, models[0].ID))
	assertEqual(t, cursor.GetNextBefore(), pbc.FormatCursorString(models[0].SubID, models[0].ID))
	assertEqual(t, *cursor.BuildNextPagingUrls(url), pageboy.CursorPagingUrls{
		Next: "https://example.com/users?a=1" +
			"&before=" + string(pbc.FormatCursorString(models[0].SubID, models[0].ID)) +
			"&limit=1",
	})

	cursor = &pageboy.Cursor{
		Before: cursor.GetNextBefore(),
		Limit:  2,
	}
	url = buildURL(cursor, *baseURL)
	assertNoError(t, db.Scopes(cursor.Paginate("SubID", "ID").Order(DESC, ASC).Scope()).Find(&models).Error)
	assertEqual(t, len(models), 2)
	assertEqual(t, models[0].ID, model3.ID)
	assertEqual(t, models[1].ID, model2.ID)
	assertEqual(t, cursor.GetNextAfter(), pbc.FormatCursorString(models[0].SubID, models[0].ID))
	assertEqual(t, cursor.GetNextBefore(), pbc.FormatCursorString(models[1].SubID, models[1].ID))
	assertEqual(t, *cursor.BuildNextPagingUrls(url), pageboy.CursorPagingUrls{
		Next: "https://example.com/users?a=1" +
			"&before=" + string(pbc.FormatCursorString(models[1].SubID, models[1].ID)) +
			"&limit=2",
	})

	cursor = &pageboy.Cursor{
		Before: cursor.GetNextBefore(),
		Limit:  2,
	}
	url = buildURL(cursor, *baseURL)
	assertNoError(t, db.Scopes(cursor.Paginate("SubID", "ID").Order(DESC, ASC).Scope()).Find(&models).Error)
	assertEqual(t, len(models), 2)
	assertEqual(t, models[0].ID, model4.ID)
	assertEqual(t, models[1].ID, model5.ID)
	assertEqual(t, cursor.GetNextAfter(), pbc.FormatCursorString(models[0].SubID, models[0].ID))
	assertEqual(t, cursor.GetNextBefore(), pbc.FormatCursorString(models[1].SubID, models[1].ID))
	assertEqual(t, *cursor.BuildNextPagingUrls(url), pageboy.CursorPagingUrls{
		Next: "",
	})
}

func TestCursorPaginateWithNullableAscDesc(t *testing.T) {
	db := openDB().Debug()
	assertNoError(t, db.Migrator().DropTable(&cursorModel{}))
	assertNoError(t, db.AutoMigrate(&cursorModel{}))

	baseURL, err := url.Parse("https://example.com/users?a=1")
	assertNoError(t, err)

	create := func(subid uint) *cursorModel {
		var s *uint
		if subid != 0 {
			s = &subid
		}
		model := &cursorModel{
			SubID: s,
		}
		assertNoError(t, db.Create(model).Error)
		return model
	}

	model1 := create(3)
	model2 := create(0)
	model3 := create(2)
	model4 := create(0)
	model5 := create(0)

	var models []*cursorModel
	cursor := &pageboy.Cursor{
		Limit: 1,
	}
	url := buildURL(cursor, *baseURL)
	assertNoError(t, db.Scopes(cursor.Paginate("SubID", "ID").Order(ASC, DESC).Scope()).Find(&models).Error)
	assertEqual(t, len(models), 1)
	assertEqual(t, models[0].ID, model5.ID)
	assertEqual(t, cursor.GetNextBefore(), pbc.FormatCursorString(models[0].SubID, models[0].ID))
	assertEqual(t, cursor.GetNextAfter(), pbc.FormatCursorString(models[0].SubID, models[0].ID))
	assertEqual(t, *cursor.BuildNextPagingUrls(url), pageboy.CursorPagingUrls{
		Next: "https://example.com/users?a=1" +
			"&after=" + string(pbc.FormatCursorString(models[0].SubID, models[0].ID)) +
			"&limit=1",
	})

	cursor = &pageboy.Cursor{
		After: cursor.GetNextAfter(),
		Limit: 2,
	}
	url = buildURL(cursor, *baseURL)
	assertNoError(t, db.Scopes(cursor.Paginate("SubID", "ID").Order(ASC, DESC).Scope()).Find(&models).Error)
	assertEqual(t, len(models), 2)
	assertEqual(t, models[0].ID, model4.ID)
	assertEqual(t, models[1].ID, model2.ID)
	assertEqual(t, cursor.GetNextBefore(), pbc.FormatCursorString(models[0].SubID, models[0].ID))
	assertEqual(t, cursor.GetNextAfter(), pbc.FormatCursorString(models[1].SubID, models[1].ID))
	assertEqual(t, *cursor.BuildNextPagingUrls(url), pageboy.CursorPagingUrls{
		Next: "https://example.com/users?a=1" +
			"&after=" + string(pbc.FormatCursorString(models[1].SubID, models[1].ID)) +
			"&limit=2",
	})

	cursor = &pageboy.Cursor{
		After: cursor.GetNextAfter(),
		Limit: 2,
	}
	url = buildURL(cursor, *baseURL)
	assertNoError(t, db.Scopes(cursor.Paginate("SubID", "ID").Order(ASC, DESC).Scope()).Find(&models).Error)
	assertEqual(t, len(models), 2)
	assertEqual(t, models[0].ID, model3.ID)
	assertEqual(t, models[1].ID, model1.ID)
	assertEqual(t, cursor.GetNextBefore(), pbc.FormatCursorString(models[0].SubID, models[0].ID))
	assertEqual(t, cursor.GetNextAfter(), pbc.FormatCursorString(models[1].SubID, models[1].ID))
	assertEqual(t, *cursor.BuildNextPagingUrls(url), pageboy.CursorPagingUrls{
		Next: "",
	})
}

func TestCursorPostgresWithoutNullsFirstLast(t *testing.T) {
	db := openDB()
	if db.Dialector.Name() != "postgres" {
		t.Skipf("This test only runs for PostgresSQL")
		return
	}
	assertNoError(t, db.Migrator().DropTable(&cursorModel{}))
	assertNoError(t, db.AutoMigrate(&cursorModel{}))

	baseURL, err := url.Parse("https://example.com/users?a=1")
	assertNoError(t, err)

	create := func(subid uint) *cursorModel {
		var s *uint
		if subid != 0 {
			s = &subid
		}
		model := &cursorModel{
			SubID: s,
		}
		assertNoError(t, db.Create(model).Error)
		return model
	}

	model1 := create(3)
	model2 := create(0)
	model3 := create(2)
	model4 := create(0)
	model5 := create(0)

	var models []*cursorModel
	cursor := &pageboy.Cursor{
		Limit: 1,
	}
	url := buildURL(cursor, *baseURL)
	assertNoError(t, db.Scopes(cursor.Paginate("SubID", "ID").Order("DESC", "ASC").Scope()).Find(&models).Error)
	assertEqual(t, len(models), 1)
	assertEqual(t, models[0].ID, model2.ID)
	assertEqual(t, cursor.GetNextAfter(), pbc.FormatCursorString(models[0].SubID, models[0].ID))
	assertEqual(t, cursor.GetNextBefore(), pbc.FormatCursorString(models[0].SubID, models[0].ID))
	assertEqual(t, *cursor.BuildNextPagingUrls(url), pageboy.CursorPagingUrls{
		Next: "https://example.com/users?a=1" +
			"&before=" + string(pbc.FormatCursorString(models[0].SubID, models[0].ID)) +
			"&limit=1",
	})

	cursor = &pageboy.Cursor{
		Before: cursor.GetNextBefore(),
		Limit:  2,
	}
	url = buildURL(cursor, *baseURL)
	assertNoError(t, db.Scopes(cursor.Paginate("SubID", "ID").Order("DESC", "ASC").Scope()).Find(&models).Error)
	assertEqual(t, len(models), 2)
	assertEqual(t, models[0].ID, model4.ID)
	assertEqual(t, models[1].ID, model5.ID)
	assertEqual(t, cursor.GetNextAfter(), pbc.FormatCursorString(models[0].SubID, models[0].ID))
	assertEqual(t, cursor.GetNextBefore(), pbc.FormatCursorString(models[1].SubID, models[1].ID))
	assertEqual(t, *cursor.BuildNextPagingUrls(url), pageboy.CursorPagingUrls{
		Next: "https://example.com/users?a=1" +
			"&before=" + string(pbc.FormatCursorString(models[1].SubID, models[1].ID)) +
			"&limit=2",
	})

	cursor = &pageboy.Cursor{
		Before: cursor.GetNextBefore(),
		Limit:  2,
	}
	url = buildURL(cursor, *baseURL)
	assertNoError(t, db.Scopes(cursor.Paginate("SubID", "ID").Order("DESC", "ASC").Scope()).Find(&models).Error)
	assertEqual(t, len(models), 2)
	assertEqual(t, models[0].ID, model1.ID)
	assertEqual(t, models[1].ID, model3.ID)
	assertEqual(t, cursor.GetNextAfter(), pbc.FormatCursorString(models[0].SubID, models[0].ID))
	assertEqual(t, cursor.GetNextBefore(), pbc.FormatCursorString(models[1].SubID, models[1].ID))
	assertEqual(t, *cursor.BuildNextPagingUrls(url), pageboy.CursorPagingUrls{
		Next: "",
	})
}

func TestCursorPostgresWithNullsLast(t *testing.T) {
	db := openDB()
	if db.Dialector.Name() != "postgres" {
		t.Skipf("This test only runs for PostgresSQL")
		return
	}
	assertNoError(t, db.Migrator().DropTable(&cursorModel{}))
	assertNoError(t, db.AutoMigrate(&cursorModel{}))

	baseURL, err := url.Parse("https://example.com/users?a=1")
	assertNoError(t, err)

	create := func(subid uint) *cursorModel {
		var s *uint
		if subid != 0 {
			s = &subid
		}
		model := &cursorModel{
			SubID: s,
		}
		assertNoError(t, db.Create(model).Error)
		return model
	}

	model1 := create(3)
	model2 := create(0)
	model3 := create(2)
	model4 := create(0)
	model5 := create(0)

	var models []*cursorModel
	cursor := &pageboy.Cursor{
		Limit: 1,
	}
	url := buildURL(cursor, *baseURL)
	assertNoError(t, db.Scopes(cursor.Paginate("SubID", "ID").Order("DESC NULLS LAST", "ASC NULLS LAST").Scope()).Find(&models).Error)
	assertEqual(t, len(models), 1)
	assertEqual(t, models[0].ID, model1.ID)
	assertEqual(t, cursor.GetNextBefore(), pbc.FormatCursorString(models[0].SubID, models[0].ID))
	assertEqual(t, cursor.GetNextAfter(), pbc.FormatCursorString(models[0].SubID, models[0].ID))
	assertEqual(t, *cursor.BuildNextPagingUrls(url), pageboy.CursorPagingUrls{
		Next: "https://example.com/users?a=1" +
			"&before=" + string(pbc.FormatCursorString(models[0].SubID, models[0].ID)) +
			"&limit=1",
	})

	cursor = &pageboy.Cursor{
		Before: cursor.GetNextBefore(),
		Limit:  2,
	}
	url = buildURL(cursor, *baseURL)
	assertNoError(t, db.Scopes(cursor.Paginate("SubID", "ID").Order("DESC NULLS LAST", "ASC NULLS LAST").Scope()).Find(&models).Error)
	assertEqual(t, len(models), 2)
	assertEqual(t, models[0].ID, model3.ID)
	assertEqual(t, models[1].ID, model2.ID)
	assertEqual(t, cursor.GetNextBefore(), pbc.FormatCursorString(models[1].SubID, models[1].ID))
	assertEqual(t, cursor.GetNextAfter(), pbc.FormatCursorString(models[0].SubID, models[0].ID))
	assertEqual(t, *cursor.BuildNextPagingUrls(url), pageboy.CursorPagingUrls{
		Next: "https://example.com/users?a=1" +
			"&before=" + string(pbc.FormatCursorString(models[1].SubID, models[1].ID)) +
			"&limit=2",
	})

	cursor = &pageboy.Cursor{
		Before: cursor.GetNextBefore(),
		Limit:  2,
	}
	url = buildURL(cursor, *baseURL)
	assertNoError(t, db.Scopes(cursor.Paginate("SubID", "ID").Order("DESC NULLS LAST", "ASC NULLS LAST").Scope()).Find(&models).Error)
	assertEqual(t, len(models), 2)
	assertEqual(t, models[0].ID, model4.ID)
	assertEqual(t, models[1].ID, model5.ID)
	assertEqual(t, cursor.GetNextBefore(), pbc.FormatCursorString(models[1].SubID, models[1].ID))
	assertEqual(t, cursor.GetNextAfter(), pbc.FormatCursorString(models[0].SubID, models[0].ID))
	assertEqual(t, *cursor.BuildNextPagingUrls(url), pageboy.CursorPagingUrls{
		Next: "",
	})
}

func TestCursorPostgresWithNullsFirst(t *testing.T) {
	db := openDB()
	if db.Dialector.Name() != "postgres" {
		t.Skipf("This test only runs for PostgresSQL")
		return
	}
	assertNoError(t, db.Migrator().DropTable(&cursorModel{}))
	assertNoError(t, db.AutoMigrate(&cursorModel{}))

	baseURL, err := url.Parse("https://example.com/users?a=1")
	assertNoError(t, err)

	create := func(subid uint) *cursorModel {
		var s *uint
		if subid != 0 {
			s = &subid
		}
		model := &cursorModel{
			SubID: s,
		}
		assertNoError(t, db.Create(model).Error)
		return model
	}

	model1 := create(3)
	model2 := create(0)
	model3 := create(2)
	model4 := create(0)
	model5 := create(0)

	var models []*cursorModel
	cursor := &pageboy.Cursor{
		Limit: 1,
	}
	url := buildURL(cursor, *baseURL)
	assertNoError(t, db.Scopes(cursor.Paginate("SubID", "ID").Order("ASC NULLS FIRST", "DESC NULLS FIRST").Scope()).Find(&models).Error)
	assertEqual(t, len(models), 1)
	assertEqual(t, models[0].ID, model5.ID)
	assertEqual(t, cursor.GetNextBefore(), pbc.FormatCursorString(models[0].SubID, models[0].ID))
	assertEqual(t, cursor.GetNextAfter(), pbc.FormatCursorString(models[0].SubID, models[0].ID))
	assertEqual(t, *cursor.BuildNextPagingUrls(url), pageboy.CursorPagingUrls{
		Next: "https://example.com/users?a=1" +
			"&after=" + string(pbc.FormatCursorString(models[0].SubID, models[0].ID)) +
			"&limit=1",
	})

	cursor = &pageboy.Cursor{
		After: cursor.GetNextAfter(),
		Limit: 2,
	}
	url = buildURL(cursor, *baseURL)
	assertNoError(t, db.Scopes(cursor.Paginate("SubID", "ID").Order("ASC NULLS FIRST", "DESC NULLS FIRST").Scope()).Find(&models).Error)
	assertEqual(t, len(models), 2)
	assertEqual(t, models[0].ID, model4.ID)
	assertEqual(t, models[1].ID, model2.ID)
	assertEqual(t, cursor.GetNextBefore(), pbc.FormatCursorString(models[0].SubID, models[0].ID))
	assertEqual(t, cursor.GetNextAfter(), pbc.FormatCursorString(models[1].SubID, models[1].ID))
	assertEqual(t, *cursor.BuildNextPagingUrls(url), pageboy.CursorPagingUrls{
		Next: "https://example.com/users?a=1" +
			"&after=" + string(pbc.FormatCursorString(models[1].SubID, models[1].ID)) +
			"&limit=2",
	})

	cursor = &pageboy.Cursor{
		After: cursor.GetNextAfter(),
		Limit: 2,
	}
	url = buildURL(cursor, *baseURL)
	assertNoError(t, db.Scopes(cursor.Paginate("SubID", "ID").Order("ASC NULLS FIRST", "DESC NULLS FIRST").Scope()).Find(&models).Error)
	assertEqual(t, len(models), 2)
	assertEqual(t, models[0].ID, model3.ID)
	assertEqual(t, models[1].ID, model1.ID)
	assertEqual(t, cursor.GetNextBefore(), pbc.FormatCursorString(models[0].SubID, models[0].ID))
	assertEqual(t, cursor.GetNextAfter(), pbc.FormatCursorString(models[1].SubID, models[1].ID))
	assertEqual(t, *cursor.BuildNextPagingUrls(url), pageboy.CursorPagingUrls{
		Next: "",
	})
}

func TestCursorPaginateReverse(t *testing.T) {
	db := openDB()
	assertNoError(t, db.Migrator().DropTable(&cursorModel{}))
	assertNoError(t, db.AutoMigrate(&cursorModel{}))

	baseURL, err := url.Parse("https://example.com/users?a=1")
	assertNoError(t, err)

	now := time.Now()

	create := func(createdAt time.Time) *cursorModel {
		model := &cursorModel{
			Model: gorm.Model{
				CreatedAt: createdAt,
			},
		}
		assertNoError(t, db.Create(model).Error)
		return model
	}

	model1 := create(now)
	model2 := create(now)
	model3 := create(now.Add(10 * time.Second))
	model4 := create(now.Add(10 * time.Hour))

	var models []*cursorModel
	cursor := &pageboy.Cursor{
		Limit:   1,
		Reverse: true,
	}
	url := buildURL(cursor, *baseURL)
	assertNoError(t, db.Scopes(cursor.Paginate("CreatedAt", "ID").Order(DESC, ASC).Scope()).Find(&models).Error)
	assertEqual(t, len(models), 1)
	assertEqual(t, models[0].ID, model2.ID)
	assertEqual(t, cursor.GetNextAfter(), pbc.FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, cursor.GetNextBefore(), pbc.FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, *cursor.BuildNextPagingUrls(url), pageboy.CursorPagingUrls{
		Next: "https://example.com/users?a=1" +
			"&after=" + string(pbc.FormatCursorString(&models[0].CreatedAt, models[0].ID)) +
			"&limit=1&reverse=true",
	})

	cursor = &pageboy.Cursor{
		After:   cursor.GetNextAfter(),
		Limit:   2,
		Reverse: true,
	}
	url = buildURL(cursor, *baseURL)
	assertNoError(t, db.Scopes(cursor.Paginate("CreatedAt", "ID").Order(DESC, ASC).Scope()).Find(&models).Error)
	assertEqual(t, len(models), 2)
	assertEqual(t, models[0].ID, model1.ID)
	assertEqual(t, models[1].ID, model3.ID)
	assertEqual(t, cursor.GetNextAfter(), pbc.FormatCursorString(&models[1].CreatedAt, models[1].ID))
	assertEqual(t, cursor.GetNextBefore(), pbc.FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, *cursor.BuildNextPagingUrls(url), pageboy.CursorPagingUrls{
		Next: "https://example.com/users?a=1" +
			"&after=" + string(pbc.FormatCursorString(&models[1].CreatedAt, models[1].ID)) +
			"&limit=2&reverse=true",
	})

	cursor = &pageboy.Cursor{
		After:   cursor.GetNextAfter(),
		Limit:   2,
		Reverse: true,
	}
	url = buildURL(cursor, *baseURL)
	assertNoError(t, db.Scopes(cursor.Paginate("CreatedAt", "ID").Order(DESC, ASC).Scope()).Find(&models).Error)
	assertEqual(t, len(models), 1)
	assertEqual(t, models[0].ID, model4.ID)
	assertEqual(t, cursor.GetNextAfter(), pbc.FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, cursor.GetNextBefore(), pbc.FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, *cursor.BuildNextPagingUrls(url), pageboy.CursorPagingUrls{
		Next: "",
	})
}

func TestCursor_where_clause_is_ambiguous(t *testing.T) {
	db := openDB()
	assertNoError(t, db.Migrator().DropTable(&cursorModel{}))
	assertNoError(t, db.Migrator().DropTable(&childModel{}))
	assertNoError(t, db.AutoMigrate(&cursorModel{}))
	assertNoError(t, db.AutoMigrate(&childModel{}))

	baseURL, err := url.Parse("https://example.com/users?a=1")
	assertNoError(t, err)

	now := time.Now()
	var childModels [2]childModel
	for i := 0; i < 2; i++ {
		assertNoError(t, db.Create(&childModels[i]).Error)
	}

	create := func(createdAt time.Time) *cursorModel {
		model := &cursorModel{
			SubID: &childModels[rand.Intn(2)].ID,
			Model: gorm.Model{
				CreatedAt: createdAt,
			},
		}
		assertNoError(t, db.Create(model).Error)
		return model
	}
	model1 := create(now)
	model2 := create(now)
	model3 := create(now.Add(2 * time.Second))
	model4 := create(now.Add(4 * time.Second))

	whereClause := func(db *gorm.DB) *gorm.DB {
		return db.Where("cursor_models.ID < ?", 999).Joins("INNER JOIN child_models as ch ON ch.id = cursor_models.sub_id")
	}

	var models []*cursorModel
	cursor := &pageboy.Cursor{
		Limit: 1,
	}
	url := buildURL(cursor, *baseURL)
	assertNoError(t, db.Scopes(whereClause, cursor.Paginate("CreatedAt", "ID").Order(ASC, ASC).Scope()).Find(&models).Error)
	assertEqual(t, len(models), 1)
	assertEqual(t, models[0].ID, model1.ID)
	assertEqual(t, cursor.GetNextAfter(), pbc.FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, cursor.GetNextBefore(), pbc.FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, *cursor.BuildNextPagingUrls(url), pageboy.CursorPagingUrls{
		Next: "https://example.com/users?a=1" +
			"&after=" + string(pbc.FormatCursorString(&models[0].CreatedAt, models[0].ID)) +
			"&limit=1",
	})

	cursor = &pageboy.Cursor{
		After: cursor.GetNextAfter(),
		Limit: 2,
	}
	url = buildURL(cursor, *baseURL)
	assertNoError(t, db.Scopes(whereClause, cursor.Paginate("CreatedAt", "ID").Order(ASC, ASC).Scope()).Find(&models).Error)
	assertEqual(t, len(models), 2)
	assertEqual(t, models[0].ID, model2.ID)
	assertEqual(t, models[1].ID, model3.ID)
	assertEqual(t, cursor.GetNextAfter(), pbc.FormatCursorString(&models[1].CreatedAt, models[1].ID))
	assertEqual(t, cursor.GetNextBefore(), pbc.FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, *cursor.BuildNextPagingUrls(url), pageboy.CursorPagingUrls{
		Next: "https://example.com/users?a=1" +
			"&after=" + string(pbc.FormatCursorString(&models[1].CreatedAt, models[1].ID)) +
			"&limit=2",
	})

	cursor = &pageboy.Cursor{
		After: cursor.GetNextAfter(),
		Limit: 2,
	}
	url = buildURL(cursor, *baseURL)
	assertNoError(t, db.Scopes(whereClause, cursor.Paginate("CreatedAt", "ID").Order(ASC, ASC).Scope()).Find(&models).Error)
	assertEqual(t, len(models), 1)
	assertEqual(t, models[0].ID, model4.ID)
	assertEqual(t, cursor.GetNextAfter(), pbc.FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, cursor.GetNextBefore(), pbc.FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, *cursor.BuildNextPagingUrls(url), pageboy.CursorPagingUrls{
		Next: "",
	})
}

func ExampleCursor() {
	db := openDB()

	type User struct {
		gorm.Model
		Name string
		Age  int
	}

	db.Migrator().DropTable(&User{})
	db.AutoMigrate(&User{})

	db.Create(&User{Name: "Alice", Age: 18})
	db.Create(&User{Name: "Bob", Age: 22})
	db.Create(&User{Name: "Carol", Age: 15})

	// Get request url.
	url, _ := url.Parse("https://localhost/path?q=%E3%81%AF%E3%82%8D%E3%83%BC")

	// Default Values. You can also use `NewCursor()`.
	cursor := &pageboy.Cursor{Limit: 2, Reverse: false}

	// Update values from a http request.

	// Fetch Records.
	var users []User
	db.Scopes(cursor.Paginate("Age", "ID").Order(ASC, DESC).Scope()).Find(&users)

	fmt.Printf("len(users) == %d\n", len(users))
	fmt.Printf("users[0].Name == \"%s\"\n", users[0].Name)
	fmt.Printf("users[1].Name == \"%s\"\n", users[1].Name)

	// Return the paging.
	j, _ := json.Marshal(cursor.BuildNextPagingUrls(url))
	fmt.Println(string(j))

	// Output:
	// len(users) == 2
	// users[0].Name == "Carol"
	// users[1].Name == "Alice"
	// {"next":"https://localhost/path?after=18_1\u0026q=%E3%81%AF%E3%82%8D%E3%83%BC"}
}
