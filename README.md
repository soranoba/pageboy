magion
==========

`magion` is a gorM pAGInatiON library.

## Features

- It support before/after pagination with GORM
- It support page/per pagination with GORM

## Usage

### Cursor

```go
var models []*Model
if db.Scopes(cursor.Paginate("CreatedAt", "ID")).Limit(10).Find(&models).Error != nil {
  return
}
```

### Pager

```go
var models []*Model
if db.Scopes(pager.Paginate()).Order("id ASC").Find(&models).Error != nil {
  return
}
```
