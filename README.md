pageboy
==========
[![CircleCI](https://circleci.com/gh/soranoba/pageboy.svg?style=svg&circle-token=977b6c270d30867fe12a0e65d34f8adbb3d7d7f2)](https://circleci.com/gh/soranoba/pageboy)
[![Go Report Card](https://goreportcard.com/badge/github.com/soranoba/pageboy)](https://goreportcard.com/report/github.com/soranoba/pageboy)
[![GoDoc](https://godoc.org/github.com/soranoba/pageboy?status.svg)](https://godoc.org/github.com/soranoba/pageboy)

`pageboy` is a [GORM](https://github.com/jinzhu/gorm) pagination library.

## Features

- It support before/after (timebase) pagination with GORM
- It support page/per pagination with GORM

## Usage

### Cursor

Cursor is sort using by time.<br>
If you sort using CreatedAt, you can prevent duplicate elements from occurring.

#### Query Formats

It is Unix Timestamp based, so the query can be specified by the user.

- Unix Timestamp in seconds
  - `https://example.com/api/users?before=1585706584&limit=10`
- Unix Timestamp in milliseconds (Depends on settings on your database)
  - `https://example.com/api/users?before=1585706584.25&limit=10`
- Unix Timestamp and Sub-Element (e.g. ID)
  - `https://example.com/api/users?before=1585706584.25_20&limit=10`

#### Index Settings

You should create an index when using a Cursor.<br>
Example using CreatedAt and ID for sorting:

```sql
CREATE INDEX created_at_id ON `users` (`created_at` DESC, `id` DESC);
```

#### Usage in Codes

```go
type UsersRequest struct {
	pageboy.Cursor
}

func getUsers(ctx echo.Context) error {
	// Set to Default Limit
	req := &UsersRequest{Cursor: pageboy.Cursor{Limit: 10, Order: pageboy.DESC}}
	// Read from query or body
	if err := ctx.Bind(req); err != nil {
		return err
	}
	// Validation
	if err := req.Validate(); err != nil {
		return err
	}
	// Read from DB
	var users []*User
	if err := db.Scopes(req.Cursor.Paginate("CreatedAt", "ID")).Find(&users).Error; err != nil {
		return err
	}
}
```

### Pager

Pager is the most basic way to specify a page size and a page number.

#### Query Formats

Page is the 1-Based number.

- `https://example.com/users?page=1&per_page=10`

#### Usage in Codes

```go
type UsersRequest struct {
	pageboy.Pager
}

func getUsers(ctx echo.Context) error {
	// Set to Default
	req := &UsersRequest{Pager: pageboy.Pager{Page: 1, PerPage: 10}}
	// Read from query or body
	if err := ctx.Bind(req); err != nil {
		return err
	}
	// Validation
	if err := req.Validate(); err != nil {
		return err
	}
	// Read from DB
	var users []*User
	if err := db.Scopes(req.Pager.Paginate()).Order("id ASC").Find(&users).Error; err != nil {
		return err
	}
}
```
