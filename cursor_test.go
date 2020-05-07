package pageboy

import (
	"testing"
	"time"

	"github.com/jinzhu/gorm"
)

type cursorModel struct {
	gorm.Model
	Name string
	Time *time.Time
}

func TestFormatCursorString(t *testing.T) {
	var id1 uint = 20
	var id2 int = 10

	format := "2006-01-02T15:04:05.999"
	ti, err := time.Parse(format, "2020-04-01T02:03:04.250")
	assertNoError(t, err)
	assertEqual(t, FormatCursorString(&ti), "1585706584.25")
	assertEqual(t, FormatCursorString(&ti, id1), "1585706584.25_20")
	assertEqual(t, FormatCursorString(&ti, &id1), "1585706584.25_20")
	assertEqual(t, FormatCursorString(&ti, id1, id2), "1585706584.25_20_10")

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
	format := "2006-01-02T15:04:05.999"
	ti, err := time.Parse(format, "2020-04-01T02:03:04.250")
	assertNoError(t, err)

	gt, ga := ParseCursorString("1585706584.25")
	assertEqual(t, gt.UnixNano(), ti.UnixNano())
	assertEqual(t, len(ga), 0)

	gt, ga = ParseCursorString("1585706584.25_20")
	assertEqual(t, gt.UnixNano(), ti.UnixNano())
	assertEqual(t, len(ga), 1)
	assertEqual(t, ga[0], int64(20))

	ti, err = time.Parse(format, "2020-04-01T02:03:04")
	assertNoError(t, err)

	gt, ga = ParseCursorString("1585706584")
	assertEqual(t, gt.UnixNano(), ti.UnixNano())
	assertEqual(t, len(ga), 0)

	gt, ga = ParseCursorString("1585706584_20")
	assertEqual(t, gt.UnixNano(), ti.UnixNano())
	assertEqual(t, len(ga), 1)
	assertEqual(t, ga[0], int64(20))
}

func TestCursorValidate(t *testing.T) {
	cursor := &Cursor{Before: "1585706584", After: "1585706584", Limit: 10}
	assertError(t, cursor.Validate())

	cursor = &Cursor{Before: "aaa", After: "", Limit: 10}
	assertError(t, cursor.Validate())

	cursor = &Cursor{Before: "", After: "aaa", Limit: 10}
	assertError(t, cursor.Validate())

	cursor = &Cursor{Before: "1585706584.25_20", After: "", Limit: 10}
	assertNoError(t, cursor.Validate())

	cursor = &Cursor{Before: "", After: "1585706584.25_20", Limit: 10}
	assertNoError(t, cursor.Validate())

	cursor = &Cursor{Before: "1585706584", After: "1585706584"}
	assertError(t, cursor.Validate())

	cursor = &Cursor{Before: "1585706584", After: "1585706584", Limit: -1}
	assertError(t, cursor.Validate())
}

func TestCursorPaginate(t *testing.T) {
	db := openDB()
	assertNoError(t, db.DropTableIfExists(&cursorModel{}).Error)
	assertNoError(t, db.AutoMigrate(&cursorModel{}).Error)

	now := time.Now()

	model1 := &cursorModel{
		Model: gorm.Model{
			CreatedAt: now,
		},
		Name: "aaa",
	}
	assertNoError(t, db.Create(&model1).Error)

	model2 := &cursorModel{
		Model: gorm.Model{
			CreatedAt: now,
		},
		Name: "bbb",
	}
	assertNoError(t, db.Create(&model2).Error)

	model3 := &cursorModel{
		Model: gorm.Model{
			CreatedAt: now.Add(10 * time.Millisecond),
		},
		Name: "ccc",
	}
	assertNoError(t, db.Create(&model3).Error)

	model4 := &cursorModel{
		Model: gorm.Model{
			CreatedAt: now.Add(10 * time.Hour),
		},
		Name: "ddd",
	}
	assertNoError(t, db.Create(&model4).Error)

	var models []*cursorModel
	cursor := &Cursor{
		Limit: 1,
	}
	assertNoError(t, db.Scopes(cursor.Paginate("CreatedAt", "ID")).Find(&models).Error)
	assertEqual(t, len(models), 1)
	assertEqual(t, models[0].ID, model4.ID)
	assertEqual(t, *cursor.GetNextAfter(), FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, *cursor.GetNextBefore(), FormatCursorString(&models[0].CreatedAt, models[0].ID))

	cursor = &Cursor{
		Before: *cursor.GetNextBefore(),
		Limit:  1, // will be overwritten
	}
	assertNoError(t, db.Scopes(cursor.Paginate("CreatedAt", "ID")).Limit(2).Find(&models).Error)
	assertEqual(t, len(models), 2)
	assertEqual(t, models[0].ID, model3.ID)
	assertEqual(t, models[1].ID, model2.ID)
	assertEqual(t, *cursor.GetNextAfter(), FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, *cursor.GetNextBefore(), FormatCursorString(&models[1].CreatedAt, models[1].ID))

	cursor = &Cursor{
		Before: *cursor.GetNextBefore(),
		Limit:  1,
	}
	assertNoError(t, db.Scopes(cursor.Paginate("CreatedAt", "ID")).Find(&models).Error)
	assertEqual(t, len(models), 1)
	assertEqual(t, models[0].ID, model1.ID)
	assertEqual(t, *cursor.GetNextAfter(), FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, cursor.GetNextBefore(), (*string)(nil))

	cursor = &Cursor{
		After: *cursor.GetNextAfter(),
		Limit: 2,
	}
	assertNoError(t, db.Scopes(cursor.Paginate("CreatedAt", "ID")).Find(&models).Error)
	assertEqual(t, len(models), 2)
	assertEqual(t, models[0].ID, model3.ID)
	assertEqual(t, models[1].ID, model2.ID)
	assertEqual(t, *cursor.GetNextAfter(), FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, *cursor.GetNextBefore(), FormatCursorString(&models[1].CreatedAt, models[1].ID))

	cursor = &Cursor{
		After: *cursor.GetNextAfter(),
		Limit: 2,
	}
	assertNoError(t, db.Scopes(cursor.Paginate("CreatedAt", "ID")).Find(&models).Error)
	assertEqual(t, len(models), 1)
	assertEqual(t, models[0].ID, model4.ID)
	assertEqual(t, *cursor.GetNextAfter(), FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, *cursor.GetNextBefore(), FormatCursorString(&models[0].CreatedAt, models[0].ID))
}

func TestCursorPaginateNullable(t *testing.T) {
	db := openDB().Debug()
	assertNoError(t, db.DropTableIfExists(&cursorModel{}).Error)
	assertNoError(t, db.AutoMigrate(&cursorModel{}).Error)

	now := time.Now()

	model1 := &cursorModel{
		Name: "aaa",
		Time: nil,
	}
	assertNoError(t, db.Create(&model1).Error)

	model2 := &cursorModel{
		Name: "bbb",
		Time: &now,
	}
	assertNoError(t, db.Create(&model2).Error)

	model3Time := now.Add(10 * time.Second)
	model3 := &cursorModel{
		Name: "ccc",
		Time: &model3Time,
	}
	assertNoError(t, db.Create(&model3).Error)

	var models []*cursorModel
	cursor := &Cursor{
		Limit: 1,
	}
	assertNoError(t, db.Scopes(cursor.Paginate("Time", "ID")).Find(&models).Error)
	assertEqual(t, len(models), 1)
	assertEqual(t, models[0].ID, model3.ID)
	assertEqual(t, *cursor.GetNextAfter(), FormatCursorString(models[0].Time, model3.ID))
	assertEqual(t, *cursor.GetNextBefore(), FormatCursorString(models[0].Time, model3.ID))

	cursor = &Cursor{
		Before: *cursor.GetNextBefore(),
		Limit:  2,
	}
	assertNoError(t, db.Scopes(cursor.Paginate("Time", "ID")).Find(&models).Error)
	assertEqual(t, len(models), 2)
	assertEqual(t, models[0].ID, model2.ID)
	assertEqual(t, models[1].ID, model1.ID)
	assertEqual(t, *cursor.GetNextAfter(), FormatCursorString(&models[0].CreatedAt, models[0].ID))
	assertEqual(t, cursor.GetNextBefore(), (*string)(nil))

	cursor = &Cursor{
		After: *cursor.GetNextAfter(),
		Limit: 2,
	}
	assertNoError(t, db.Scopes(cursor.Paginate("Time", "ID")).Find(&models).Error)
	assertEqual(t, len(models), 1)
	assertEqual(t, models[0].ID, model3.ID)
	assertEqual(t, *cursor.GetNextAfter(), FormatCursorString(models[0].Time, models[0].ID))
	assertEqual(t, *cursor.GetNextBefore(), FormatCursorString(models[0].Time, models[0].ID))
}
