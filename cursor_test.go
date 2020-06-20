package pageboy

import (
	"net/url"
	"strconv"
	"testing"
	"time"

	"gorm.io/gorm"
)

type cursorModel struct {
	gorm.Model
	Name string
	Time *time.Time
}

func (cursor *Cursor) buildURL(base url.URL) *url.URL {
	query := base.Query()
	if cursor.Before != "" {
		query.Del("before")
		query.Add("before", cursor.Before)
	}
	if cursor.After != "" {
		query.Del("after")
		query.Add("after", cursor.After)
	}
	if cursor.Limit != 0 {
		query.Del("limit")
		query.Add("limit", strconv.Itoa(cursor.Limit))
	}
	if cursor.Order != "" {
		query.Del("order")
		query.Add("order", string(cursor.Order))
	}

	base.RawQuery = query.Encode()
	return &base
}

func TestFormatCursorString(t *testing.T) {
	var id1 uint = 20
	var id2 int = 10

	format := "2006-01-02T15:04:05.9999"
	ti, err := time.Parse(format, "2020-04-01T02:03:04.0250")
	assertNoError(t, err)
	assertEqual(t, FormatCursorString(&ti), "1585706584.025")
	assertEqual(t, FormatCursorString(&ti, id1), "1585706584.025_20")
	assertEqual(t, FormatCursorString(&ti, &id1), "1585706584.025_20")
	assertEqual(t, FormatCursorString(&ti, id1, id2), "1585706584.025_20_10")

	ti, err = time.Parse(format, "2020-04-01T02:03:04")
	assertNoError(t, err)
	assertEqual(t, FormatCursorString(&ti), "1585706584")
	assertEqual(t, FormatCursorString(&ti, id1), "1585706584_20")
	assertEqual(t, FormatCursorString(&ti, &id1), "1585706584_20")

	assertEqual(t, FormatCursorString(nil, nil), "_")
	assertEqual(t, FormatCursorString(nil, nil, nil), "__")
	assertEqual(t, FormatCursorString(nil, id1), "_20")
}

func TestValidateCursorString(t *testing.T) {
	assertEqual(t, ValidateCursorString("1585706584"), true)
	assertEqual(t, ValidateCursorString("1585706584.25"), true)
	assertEqual(t, ValidateCursorString("1585706584.250"), true)
	assertEqual(t, ValidateCursorString("1585706584.25_20"), true)
	assertEqual(t, ValidateCursorString("1585706584_20"), true)
	assertEqual(t, ValidateCursorString("1585706_584.25"), true)
	assertEqual(t, ValidateCursorString("1585706_584_25"), true)

	assertEqual(t, ValidateCursorString("15857065.84.25"), false)
	assertEqual(t, ValidateCursorString("1585706aa4.25"), false)
	assertEqual(t, ValidateCursorString("1585706584.2aa"), false)
}

func TestParseCursorString(t *testing.T) {
	format := "2006-01-02T15:04:05.9999"
	ti, err := time.Parse(format, "2020-04-01T02:03:04.0250")
	assertNoError(t, err)

	ga := ParseCursorString("1585706584.025")
	assertEqual(t, len(ga), 1)
	assertEqual(t, ga[0].Time().UnixNano(), ti.UnixNano())

	ga = ParseCursorString("1585706584.025_20")
	assertEqual(t, len(ga), 2)
	assertEqual(t, ga[0].Time().UnixNano(), ti.UnixNano())
	assertEqual(t, ga[1].Int64(), int64(20))

	ti, err = time.Parse(format, "2020-04-01T02:03:04")
	assertNoError(t, err)

	ga = ParseCursorString("1585706584")
	assertEqual(t, len(ga), 1)
	assertEqual(t, ga[0].Int64(), int64(1585706584))

	ga = ParseCursorString("1585706584_20")
	assertEqual(t, len(ga), 2)
	assertEqual(t, ga[0].Time().UnixNano(), ti.UnixNano())
	assertEqual(t, ga[1].Int64(), int64(20))

	ga = ParseCursorString("_1")
	assertEqual(t, len(ga), 2)
	assertEqual(t, ga[0].Time(), (*time.Time)(nil))
	assertEqual(t, ga[1].Int64(), int64(1))

	ga = ParseCursorString("_1__2")
	assertEqual(t, len(ga), 4)
	assertEqual(t, ga[0].Time(), (*time.Time)(nil))
	assertEqual(t, ga[1].Int64(), int64(1))
	assertEqual(t, ga[2].IsNil(), true)
	assertEqual(t, ga[3].Int64(), int64(2))
}

func TestCursorValidate(t *testing.T) {
	// invalid before params
	cursor := &Cursor{Before: "aaa", After: "", Limit: 10, Order: DESC}
	assertError(t, cursor.Validate())

	// invalid after params
	cursor = &Cursor{Before: "", After: "aaa", Limit: 10, Order: DESC}
	assertError(t, cursor.Validate())

	// invalid limit params
	cursor = &Cursor{Before: "1585706584", After: "1585706584", Order: DESC}
	assertError(t, cursor.Validate())
	cursor = &Cursor{Before: "1585706584", After: "1585706584", Limit: -1, Order: DESC}
	assertError(t, cursor.Validate())

	cursor = &Cursor{Before: "1585706584.025_20", After: "", Limit: 10, Order: DESC}
	assertNoError(t, cursor.Validate())

	cursor = &Cursor{Before: "", After: "1585706584.025_20", Limit: 10, Order: DESC}
	assertNoError(t, cursor.Validate())

	cursor = &Cursor{Before: "1585706584", After: "1585706584", Limit: 10, Order: DESC}
	assertNoError(t, cursor.Validate())

	cursor = &Cursor{Before: "", After: "", Limit: 10, Order: DESC}
	assertNoError(t, cursor.Validate())

	cursor = &Cursor{Before: "", After: "", Limit: 10, Order: ASC}
	assertNoError(t, cursor.Validate())
}

func TestCursorPaginateDESC(t *testing.T) {
	db := openDB()
	assertNoError(t, db.Migrator().DropTable(&cursorModel{}))
	assertNoError(t, db.AutoMigrate(&cursorModel{}))

	baseUrl, err := url.Parse("https://example.com/users?a=1")
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
	cursor := &Cursor{
		Order: DESC,
		Limit: 1,
	}
	url := cursor.buildURL(*baseUrl)
	assertNoError(t, db.Scopes(cursor.Paginate("CreatedAt", "ID")).Find(&models).Error)
	assertEqual(t, len(models), 1)
	assertEqual(t, models[0].ID, model4.ID)
	assertEqual(t, cursor.GetNextAfter(), FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, cursor.GetNextBefore(), FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, *cursor.BuildNextPagingUrls(url), CursorPagingUrls{
		Before: "https://example.com/users?a=1" +
			"&before=" + FormatCursorString(&models[0].CreatedAt, models[0].ID) +
			"&limit=1&order=desc",
		After: "",
	})

	cursor = &Cursor{
		Before: cursor.GetNextBefore(),
		Order:  DESC,
		Limit:  2,
	}
	url = cursor.buildURL(*baseUrl)
	assertNoError(t, db.Scopes(cursor.Paginate("CreatedAt", "ID")).Find(&models).Error)
	assertEqual(t, len(models), 2)
	assertEqual(t, models[0].ID, model3.ID)
	assertEqual(t, models[1].ID, model2.ID)
	assertEqual(t, cursor.GetNextAfter(), FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, cursor.GetNextBefore(), FormatCursorString(&models[1].CreatedAt, models[1].ID))
	assertEqual(t, *cursor.BuildNextPagingUrls(url), CursorPagingUrls{
		Before: "https://example.com/users?a=1" +
			"&before=" + FormatCursorString(&models[1].CreatedAt, models[1].ID) +
			"&limit=2&order=desc",
		After: "",
	})

	cursor = &Cursor{
		Before: cursor.GetNextBefore(),
		Order:  DESC,
		Limit:  2,
	}
	url = cursor.buildURL(*baseUrl)
	assertNoError(t, db.Scopes(cursor.Paginate("CreatedAt", "ID")).Find(&models).Error)
	assertEqual(t, len(models), 1)
	assertEqual(t, models[0].ID, model1.ID)
	assertEqual(t, cursor.GetNextAfter(), FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, cursor.GetNextBefore(), FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, *cursor.BuildNextPagingUrls(url), CursorPagingUrls{
		Before: "",
		After:  "",
	})
}

func TestCursorPaginateASC(t *testing.T) {
	db := openDB()
	assertNoError(t, db.Migrator().DropTable(&cursorModel{}))
	assertNoError(t, db.AutoMigrate(&cursorModel{}))

	baseUrl, err := url.Parse("https://example.com/users?a=1")
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
	cursor := &Cursor{
		Order: ASC,
		Limit: 1,
	}
	url := cursor.buildURL(*baseUrl)
	assertNoError(t, db.Scopes(cursor.Paginate("CreatedAt", "ID")).Find(&models).Error)
	assertEqual(t, len(models), 1)
	assertEqual(t, models[0].ID, model1.ID)
	assertEqual(t, cursor.GetNextAfter(), FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, cursor.GetNextBefore(), FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, *cursor.BuildNextPagingUrls(url), CursorPagingUrls{
		Before: "",
		After: "https://example.com/users?a=1" +
			"&after=" + FormatCursorString(&models[0].CreatedAt, models[0].ID) +
			"&limit=1&order=asc",
	})

	cursor = &Cursor{
		After: cursor.GetNextAfter(),
		Order: ASC,
		Limit: 2,
	}
	url = cursor.buildURL(*baseUrl)
	assertNoError(t, db.Scopes(cursor.Paginate("CreatedAt", "ID")).Find(&models).Error)
	assertEqual(t, len(models), 2)
	assertEqual(t, models[0].ID, model2.ID)
	assertEqual(t, models[1].ID, model3.ID)
	assertEqual(t, cursor.GetNextAfter(), FormatCursorString(&models[1].CreatedAt, models[1].ID))
	assertEqual(t, cursor.GetNextBefore(), FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, *cursor.BuildNextPagingUrls(url), CursorPagingUrls{
		Before: "",
		After: "https://example.com/users?a=1" +
			"&after=" + FormatCursorString(&models[1].CreatedAt, models[1].ID) +
			"&limit=2&order=asc",
	})

	cursor = &Cursor{
		After: cursor.GetNextAfter(),
		Order: ASC,
		Limit: 2,
	}
	url = cursor.buildURL(*baseUrl)
	assertNoError(t, db.Scopes(cursor.Paginate("CreatedAt", "ID")).Find(&models).Error)
	assertEqual(t, len(models), 1)
	assertEqual(t, models[0].ID, model4.ID)
	assertEqual(t, cursor.GetNextAfter(), FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, cursor.GetNextBefore(), FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, *cursor.BuildNextPagingUrls(url), CursorPagingUrls{
		Before: "",
		After:  "",
	})
}

func TestCursorPaginateWithBeforeDESC(t *testing.T) {
	db := openDB()
	assertNoError(t, db.Migrator().DropTable(&cursorModel{}))
	assertNoError(t, db.AutoMigrate(&cursorModel{}))

	baseUrl, err := url.Parse("https://example.com/users?a=1")
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
	cursor := &Cursor{
		Before: FormatCursorString(&model4.CreatedAt, model4.ID),
		Order:  DESC,
		Limit:  2,
	}
	url := cursor.buildURL(*baseUrl)
	assertNoError(t, db.Scopes(cursor.Paginate("CreatedAt", "ID")).Find(&models).Error)
	assertEqual(t, len(models), 2)
	assertEqual(t, models[0].ID, model3.ID)
	assertEqual(t, models[1].ID, model2.ID)
	assertEqual(t, cursor.GetNextAfter(), FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, cursor.GetNextBefore(), FormatCursorString(&models[1].CreatedAt, models[1].ID))
	assertEqual(t, *cursor.BuildNextPagingUrls(url), CursorPagingUrls{
		Before: "https://example.com/users?a=1" +
			"&before=" + FormatCursorString(&models[1].CreatedAt, models[1].ID) +
			"&limit=2&order=desc",
		After: "",
	})

	cursor = &Cursor{
		Before: cursor.GetNextBefore(),
		Order:  DESC,
		Limit:  2,
	}
	url = cursor.buildURL(*baseUrl)
	assertNoError(t, db.Scopes(cursor.Paginate("CreatedAt", "ID")).Find(&models).Error)
	assertEqual(t, len(models), 1)
	assertEqual(t, models[0].ID, model1.ID)
	assertEqual(t, cursor.GetNextAfter(), FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, cursor.GetNextBefore(), FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, *cursor.BuildNextPagingUrls(url), CursorPagingUrls{
		Before: "",
		After:  "",
	})
}

func TestCursorPaginateWithBeforeASC(t *testing.T) {
	db := openDB()
	assertNoError(t, db.Migrator().DropTable(&cursorModel{}))
	assertNoError(t, db.AutoMigrate(&cursorModel{}))

	baseUrl, err := url.Parse("https://example.com/users?a=1")
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
	cursor := &Cursor{
		Before: FormatCursorString(&model4.CreatedAt, model4.ID),
		Order:  ASC,
		Limit:  2,
	}
	url := cursor.buildURL(*baseUrl)
	assertNoError(t, db.Scopes(cursor.Paginate("CreatedAt", "ID")).Find(&models).Error)
	assertEqual(t, len(models), 2)
	assertEqual(t, models[0].ID, model1.ID)
	assertEqual(t, models[1].ID, model2.ID)
	assertEqual(t, cursor.GetNextAfter(), FormatCursorString(&models[1].CreatedAt, models[1].ID))
	assertEqual(t, cursor.GetNextBefore(), FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, *cursor.BuildNextPagingUrls(url), CursorPagingUrls{
		Before: "",
		After: "https://example.com/users?a=1" +
			"&after=" + FormatCursorString(&models[1].CreatedAt, models[1].ID) +
			"&before=" + cursor.Before +
			"&limit=2&order=asc",
	})

	cursor = &Cursor{
		After:  cursor.GetNextAfter(),
		Before: cursor.Before,
		Order:  ASC,
		Limit:  2,
	}
	url = cursor.buildURL(*baseUrl)
	assertNoError(t, db.Scopes(cursor.Paginate("CreatedAt", "ID")).Find(&models).Error)
	assertEqual(t, len(models), 1)
	assertEqual(t, models[0].ID, model3.ID)
	assertEqual(t, cursor.GetNextAfter(), FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, cursor.GetNextBefore(), FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, *cursor.BuildNextPagingUrls(url), CursorPagingUrls{
		Before: "",
		After:  "",
	})
}

func TestCursorPaginateWithAfterDESC(t *testing.T) {
	db := openDB()
	assertNoError(t, db.Migrator().DropTable(&cursorModel{}))
	assertNoError(t, db.AutoMigrate(&cursorModel{}))

	baseUrl, err := url.Parse("https://example.com/users?a=1")
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
	cursor := &Cursor{
		After: FormatCursorString(&model1.CreatedAt, model1.ID),
		Order: DESC,
		Limit: 2,
	}
	url := cursor.buildURL(*baseUrl)
	assertNoError(t, db.Scopes(cursor.Paginate("CreatedAt", "ID")).Find(&models).Error)
	assertEqual(t, len(models), 2)
	assertEqual(t, models[0].ID, model4.ID)
	assertEqual(t, models[1].ID, model3.ID)
	assertEqual(t, cursor.GetNextAfter(), FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, cursor.GetNextBefore(), FormatCursorString(&models[1].CreatedAt, models[1].ID))
	assertEqual(t, *cursor.BuildNextPagingUrls(url), CursorPagingUrls{
		Before: "https://example.com/users?a=1" +
			"&after=" + cursor.After +
			"&before=" + FormatCursorString(&models[1].CreatedAt, models[1].ID) +
			"&limit=2&order=desc",
		After: "",
	})

	cursor = &Cursor{
		After:  cursor.After,
		Before: cursor.GetNextBefore(),
		Order:  DESC,
		Limit:  2,
	}
	url = cursor.buildURL(*baseUrl)
	assertNoError(t, db.Scopes(cursor.Paginate("CreatedAt", "ID")).Find(&models).Error)
	assertEqual(t, len(models), 1)
	assertEqual(t, models[0].ID, model2.ID)
	assertEqual(t, cursor.GetNextAfter(), FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, cursor.GetNextBefore(), FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, *cursor.BuildNextPagingUrls(url), CursorPagingUrls{
		Before: "",
		After:  "",
	})
}

func TestCursorPaginateWithAfterASC(t *testing.T) {
	db := openDB()
	assertNoError(t, db.Migrator().DropTable(&cursorModel{}))
	assertNoError(t, db.AutoMigrate(&cursorModel{}))

	baseUrl, err := url.Parse("https://example.com/users?a=1")
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
	cursor := &Cursor{
		After: FormatCursorString(&model1.CreatedAt, model1.ID),
		Order: ASC,
		Limit: 2,
	}
	url := cursor.buildURL(*baseUrl)
	assertNoError(t, db.Scopes(cursor.Paginate("CreatedAt", "ID")).Find(&models).Error)
	assertEqual(t, len(models), 2)
	assertEqual(t, models[0].ID, model2.ID)
	assertEqual(t, models[1].ID, model3.ID)
	assertEqual(t, cursor.GetNextAfter(), FormatCursorString(&models[1].CreatedAt, models[1].ID))
	assertEqual(t, cursor.GetNextBefore(), FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, *cursor.BuildNextPagingUrls(url), CursorPagingUrls{
		Before: "",
		After: "https://example.com/users?a=1" +
			"&after=" + FormatCursorString(&models[1].CreatedAt, models[1].ID) +
			"&limit=2&order=asc",
	})

	cursor = &Cursor{
		After: cursor.GetNextAfter(),
		Order: ASC,
		Limit: 2,
	}
	url = cursor.buildURL(*baseUrl)
	assertNoError(t, db.Scopes(cursor.Paginate("CreatedAt", "ID")).Find(&models).Error)
	assertEqual(t, len(models), 1)
	assertEqual(t, models[0].ID, model4.ID)
	assertEqual(t, cursor.GetNextAfter(), FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, cursor.GetNextBefore(), FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, *cursor.BuildNextPagingUrls(url), CursorPagingUrls{
		Before: "",
		After:  "",
	})
}

func TestCursorPaginateWithAfterAndBeforeDESC(t *testing.T) {
	db := openDB()
	assertNoError(t, db.Migrator().DropTable(&cursorModel{}))
	assertNoError(t, db.AutoMigrate(&cursorModel{}))

	baseUrl, err := url.Parse("https://example.com/users?a=1")
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
	cursor := &Cursor{
		Before: FormatCursorString(&model5.CreatedAt, model5.ID),
		After:  FormatCursorString(&model1.CreatedAt, model1.ID),
		Order:  DESC,
		Limit:  2,
	}
	url := cursor.buildURL(*baseUrl)
	assertNoError(t, db.Scopes(cursor.Paginate("CreatedAt", "ID")).Find(&models).Error)
	assertEqual(t, len(models), 2)
	assertEqual(t, models[0].ID, model4.ID)
	assertEqual(t, models[1].ID, model3.ID)
	assertEqual(t, cursor.GetNextAfter(), FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, cursor.GetNextBefore(), FormatCursorString(&models[1].CreatedAt, models[1].ID))
	assertEqual(t, *cursor.BuildNextPagingUrls(url), CursorPagingUrls{
		Before: "https://example.com/users?a=1" +
			"&after=" + cursor.After +
			"&before=" + FormatCursorString(&models[1].CreatedAt, models[1].ID) +
			"&limit=2&order=desc",
	})

	cursor = &Cursor{
		Before: cursor.GetNextBefore(),
		After:  cursor.After,
		Order:  DESC,
		Limit:  2,
	}
	url = cursor.buildURL(*baseUrl)
	assertNoError(t, db.Scopes(cursor.Paginate("CreatedAt", "ID")).Find(&models).Error)
	assertEqual(t, len(models), 1)
	assertEqual(t, models[0].ID, model2.ID)
	assertEqual(t, cursor.GetNextAfter(), FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, cursor.GetNextBefore(), FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, *cursor.BuildNextPagingUrls(url), CursorPagingUrls{
		Before: "",
		After:  "",
	})
}

func TestCursorPaginateWithAfterAndBeforeASC(t *testing.T) {
	db := openDB()
	assertNoError(t, db.Migrator().DropTable(&cursorModel{}))
	assertNoError(t, db.AutoMigrate(&cursorModel{}))

	baseUrl, err := url.Parse("https://example.com/users?a=1")
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
	cursor := &Cursor{
		Before: FormatCursorString(&model5.CreatedAt, model5.ID),
		After:  FormatCursorString(&model1.CreatedAt, model1.ID),
		Order:  ASC,
		Limit:  2,
	}
	url := cursor.buildURL(*baseUrl)
	assertNoError(t, db.Scopes(cursor.Paginate("CreatedAt", "ID")).Find(&models).Error)
	assertEqual(t, len(models), 2)
	assertEqual(t, models[0].ID, model2.ID)
	assertEqual(t, models[1].ID, model3.ID)
	assertEqual(t, cursor.GetNextAfter(), FormatCursorString(&models[1].CreatedAt, models[1].ID))
	assertEqual(t, cursor.GetNextBefore(), FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, *cursor.BuildNextPagingUrls(url), CursorPagingUrls{
		Before: "",
		After: "https://example.com/users?a=1" +
			"&after=" + FormatCursorString(&models[1].CreatedAt, models[1].ID) +
			"&before=" + cursor.Before +
			"&limit=2&order=asc",
	})

	cursor = &Cursor{
		Before: cursor.Before,
		After:  cursor.GetNextAfter(),
		Order:  ASC,
		Limit:  2,
	}
	url = cursor.buildURL(*baseUrl)
	assertNoError(t, db.Scopes(cursor.Paginate("CreatedAt", "ID")).Find(&models).Error)
	assertEqual(t, len(models), 1)
	assertEqual(t, models[0].ID, model4.ID)
	assertEqual(t, cursor.GetNextAfter(), FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, cursor.GetNextBefore(), FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, *cursor.BuildNextPagingUrls(url), CursorPagingUrls{
		Before: "",
		After:  "",
	})
}

func TestCursorPaginateWithEmpty(t *testing.T) {
	db := openDB()
	assertNoError(t, db.Migrator().DropTable(&cursorModel{}))
	assertNoError(t, db.AutoMigrate(&cursorModel{}))

	var models []*cursorModel
	cursor := &Cursor{
		Order: DESC,
		Limit: 1,
	}
	assertNoError(t, db.Scopes(cursor.Paginate("Time", "ID")).Find(&models).Error)
	assertEqual(t, len(models), 0)
	assertEqual(t, cursor.GetNextAfter(), "_0")
	assertEqual(t, cursor.GetNextBefore(), "_0")
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
	cursor := &Cursor{
		Order: DESC,
		Limit: 1,
	}
	assertNoError(t, db.Scopes(cursor.Paginate("Time", "ID")).Find(&models).Error)
	assertEqual(t, len(models), 1)
	assertEqual(t, models[0].ID, model4.ID)
	assertEqual(t, cursor.GetNextAfter(), FormatCursorString(models[0].Time, models[0].ID))
	assertEqual(t, cursor.GetNextBefore(), FormatCursorString(models[0].Time, models[0].ID))

	cursor = &Cursor{
		Before: cursor.GetNextBefore(),
		Order:  DESC,
		Limit:  2,
	}
	assertNoError(t, db.Scopes(cursor.Paginate("Time", "ID")).Find(&models).Error)
	assertEqual(t, len(models), 2)
	assertEqual(t, models[0].ID, model3.ID)
	assertEqual(t, models[1].ID, model2.ID)
	assertEqual(t, cursor.GetNextAfter(), FormatCursorString(models[0].Time, models[0].ID))
	assertEqual(t, cursor.GetNextBefore(), FormatCursorString(models[1].Time, models[1].ID))

	cursor = &Cursor{
		Before: cursor.GetNextBefore(),
		Order:  DESC,
		Limit:  2,
	}
	assertNoError(t, db.Scopes(cursor.Paginate("Time", "ID")).Find(&models).Error)
	assertEqual(t, len(models), 1)
	assertEqual(t, models[0].ID, model1.ID)
	assertEqual(t, cursor.GetNextAfter(), FormatCursorString(models[0].Time, models[0].ID))
	assertEqual(t, cursor.GetNextBefore(), FormatCursorString(models[0].Time, models[0].ID))
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
	cursor := &Cursor{
		Order: ASC,
		Limit: 1,
	}
	assertNoError(t, db.Scopes(cursor.Paginate("Time", "ID")).Find(&models).Error)
	assertEqual(t, len(models), 1)
	assertEqual(t, models[0].ID, model1.ID)
	assertEqual(t, cursor.GetNextBefore(), FormatCursorString(models[0].Time, models[0].ID))
	assertEqual(t, cursor.GetNextAfter(), FormatCursorString(models[0].Time, models[0].ID))

	cursor = &Cursor{
		After: cursor.GetNextAfter(),
		Order: ASC,
		Limit: 2,
	}
	assertNoError(t, db.Scopes(cursor.Paginate("Time", "ID")).Find(&models).Error)
	assertEqual(t, len(models), 2)
	assertEqual(t, models[0].ID, model2.ID)
	assertEqual(t, models[1].ID, model3.ID)
	assertEqual(t, cursor.GetNextBefore(), FormatCursorString(models[0].Time, models[0].ID))
	assertEqual(t, cursor.GetNextAfter(), FormatCursorString(models[1].Time, models[1].ID))

	cursor = &Cursor{
		After: cursor.GetNextAfter(),
		Order: ASC,
		Limit: 2,
	}
	assertNoError(t, db.Scopes(cursor.Paginate("Time", "ID")).Find(&models).Error)
	assertEqual(t, len(models), 1)
	assertEqual(t, models[0].ID, model4.ID)
	assertEqual(t, cursor.GetNextBefore(), FormatCursorString(models[0].Time, models[0].ID))
	assertEqual(t, cursor.GetNextAfter(), FormatCursorString(models[0].Time, models[0].ID))
}
