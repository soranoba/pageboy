package pageboy

import (
	"encoding/json"
	"fmt"

	"gorm.io/gorm"
)

func ExamplePager() {
	db := openDB()
	// Please execute it only once immediately after opening DB.
	RegisterCallbacks(db)

	type User struct {
		gorm.Model
		Name string
	}

	db.Migrator().DropTable(&User{})
	db.AutoMigrate(&User{})

	db.Create(&User{Name: "Alice"})
	db.Create(&User{Name: "Bob"})
	db.Create(&User{Name: "Carol"})

	// Default Values.
	pager := &Pager{Page: 1, PerPage: 2}

	// Update values from a http request.

	// Fetch Records.
	var users []User
	db.Scopes(pager.Scope()).Order("id ASC").Find(&users)

	fmt.Printf("len(users) == %d\n", len(users))
	fmt.Printf("users[0].Name == \"%s\"\n", users[0].Name)
	fmt.Printf("users[1].Name == \"%s\"\n", users[1].Name)

	// Return the Summary.
	j, _ := json.Marshal(pager.Summary())
	fmt.Println(string(j))

	// Output:
	// len(users) == 2
	// users[0].Name == "Alice"
	// users[1].Name == "Bob"
	// {"page":1,"per_page":2,"total_count":3,"total_page":2}
}
