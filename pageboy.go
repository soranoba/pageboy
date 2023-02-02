// Package pageboy is a pagination library with GORM.
//
// # Links
//
// GORM: https://github.com/go-gorm/gorm
// Source code: https://github.com/soranoba/pageboy
package pageboy

import "gorm.io/gorm"

// RegisterCallbacks register the Callback used by pageboy in gorm.DB.
// This function MUST execute only once immediately after opening the DB. (https://pkg.go.dev/gorm.io/gorm#Open)
// DO NOT execute every time you create new Session (https://pkg.go.dev/gorm.io/gorm#DB.Session).
func RegisterCallbacks(db *gorm.DB) {
	registerCursorCallbacks(db)
	registerPagerCallbacks(db)
}
