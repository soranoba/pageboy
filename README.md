pageboy
==========
[![CircleCI](https://circleci.com/gh/soranoba/pageboy.svg?style=svg&circle-token=977b6c270d30867fe12a0e65d34f8adbb3d7d7f2)](https://circleci.com/gh/soranoba/pageboy)
[![Go Report Card](https://goreportcard.com/badge/github.com/soranoba/pageboy)](https://goreportcard.com/report/github.com/soranoba/pageboy)
[![GoDoc](https://godoc.org/github.com/soranoba/pageboy?status.svg)](https://godoc.org/github.com/soranoba/pageboy)

`pageboy` is a gorM pAGInatiON library.

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
