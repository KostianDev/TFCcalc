package main

import (
	"log"
	"os"

	"fyne.io/fyne/v2/app"

	"tfccalc/data"
	"tfccalc/ui"
)

func main() {
	jsonPath := os.Getenv("TFC_ALLOYS_JSON")
	if jsonPath == "" {
		jsonPath = "./assets/alloys.json"
	}
	if err := data.InitJSON(jsonPath); err != nil {
		mysqlDSN := os.Getenv("TFC_MYSQL_DSN")
		if mysqlDSN == "" {
			log.Fatalf("Failed to initialize JSON repository: %v", err)
		}
		if err := data.InitMySQL(mysqlDSN); err != nil {
			log.Fatalf("Failed to initialize MySQL repository: %v", err)
		}
	}

	myApp := app.New()
	myWindow := ui.BuildUI(myApp)
	myWindow.ShowAndRun()
}
