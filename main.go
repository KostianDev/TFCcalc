package main

import (
	"log"
	"os"

	"fyne.io/fyne/v2/app"

	adapterui "tfccalc/adapter/ui"
	"tfccalc/data"
	"tfccalc/usecase/alloy"
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
	repo := data.Repository()
	if repo == nil {
		log.Fatal("Repository not initialized")
	}
	alloyService := alloy.NewService(repo)

	myApp := app.New()
	myWindow := adapterui.BuildUI(myApp, alloyService)
	myWindow.ShowAndRun()
}
