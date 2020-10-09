pageboy
==========
[![CircleCI](https://circleci.com/gh/soranoba/pageboy.svg?style=svg&circle-token=977b6c270d30867fe12a0e65d34f8adbb3d7d7f2)](https://circleci.com/gh/soranoba/pageboy)
[![Go Report Card](https://goreportcard.com/badge/github.com/soranoba/pageboy)](https://goreportcard.com/report/github.com/soranoba/pageboy)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/soranoba/pageboy)](https://pkg.go.dev/github.com/soranoba/pageboy)

`pageboy` is a pagination library with [GORM v2](https://github.com/go-gorm/gorm).

## Overviews

- 💪　Support both of before/after (Cursor) and page/per (Pager) DB pagination.
- 🤗　Accept human readable queries.
  - Like them: `?page=1&per_page=2` and `?before=1585706584&limit=10`
  - We can also customize it if needed.
- 💖　We can write smart code using GORM scopes.

## Installation

To install it, run:

```bash
go get -u github.com/soranoba/pageboy
```

## Usage

### Cursor

Cursor can be used to indicate a range that is after or before that value.<br>
It can sort using by time or integer.<br>
For example, when we sort using CreatedAt and ID, it can prevent duplicate values from occurring.

#### Query Formats

- Simple numbers
  - `https://example.com/api/users?before=1&limit=10`
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
CREATE INDEX created_at_id ON users (created_at DESC, id DESC);
```

#### Usage in Codes

```go
type UsersRequest struct {
	pageboy.Cursor
}

func getUsers(ctx echo.Context) error {
	// Set to Default Limit
	req := &UsersRequest{Cursor: pageboy.Cursor{Limit: 10}}
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
	if err := db.Scopes(req.Cursor.Paginate("CreatedAt", "ID").Order("DESC", "DESC").Scope()).Find(&users).Error; err != nil {
		return err
	}
}
```

### Pager

Pager can be used to indicate a range that is specified a page size and a page number.

#### Query Formats

It includes a page which is 1-Based number, and per_page.

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
	if err := db.Scopes(req.Pager.Scope()).Order("id ASC").Find(&users).Error; err != nil {
		return err
	}
}
```
