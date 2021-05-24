package pageboy

import (
	"encoding/json"
	"fmt"
	"net/url"

	"gorm.io/gorm"
)

func ExampleCursor() {
	db := openDB()
	// Please execute it only once immediately after opening DB.
	RegisterCallbacks(db)

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
	cursor := &Cursor{Limit: 2, Reverse: false}

	// Update values from a http request.

	// Fetch Records.
	var users []User
	db.Scopes(cursor.Paginate("Age", "ID").Order("ASC", "DESC").Scope()).Find(&users)

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

func ExampleCursor_Order() {
	cursor := &Cursor{Limit: 2, Reverse: false}
	// For usually
	cursor.Paginate("CreatedAt", "ID").Order("DESC", "ASC").Scope()
	// For PostgresSQL
	cursor.Paginate("CreatedAt", "ID").Order("DESC NULLS LAST", "ASC NULLS FIRST").Scope()
}
