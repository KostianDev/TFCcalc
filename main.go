// Package main is the entry point for the TFC Alloy Calculator application.
package main

import (
	"fyne.io/fyne/v2/app"
	"tfccalc/ui" // Import our UI package
)

func main() {
	// Create a new Fyne application
	myApp := app.New()
	// Build the main window using the function from the ui package
	myWindow := ui.BuildUI(myApp)
	// Show the window and run the application loop
	myWindow.ShowAndRun()
}