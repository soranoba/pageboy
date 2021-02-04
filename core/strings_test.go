package core

import "testing"

func TestToSnake(t *testing.T) {
	assertEqual(t, toSnake("CreatedAt"), "created_at")
	assertEqual(t, toSnake("ID"), "id")
	assertEqual(t, toSnake("TestID"), "test_id")
	assertEqual(t, toSnake("ImageURL"), "image_url")
	assertEqual(t, toSnake("AbCdEf"), "ab_cd_ef")
	assertEqual(t, toSnake("models.ID"), "models.id")
	assertEqual(t, toSnake("`models`.`ID`"), "`models`.`id`")
}
