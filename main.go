package main

import (
	"fmt"
	"log"
	"tfccalc/data"
	"tfccalc/ui"

	"fyne.io/fyne/v2/app"
)

func main() {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&charset=utf8mb4",
		"tfccalc_user", "tfccalc_pass", "127.0.0.1", 3306, "tfccalc_db",
	)
	if err := data.InitDB(dsn); err != nil {
		log.Fatalf("Failed to initialize DB: %v", err)
	}

	myApp := app.New()
	myWindow := ui.BuildUI(myApp)
	myWindow.ShowAndRun()
}
