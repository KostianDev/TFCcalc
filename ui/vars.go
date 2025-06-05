// ui/vars.go
package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// У цьому файлі ми зберігаємо всі глобальні змінні, що використовуються в різних
// файлах пакету ui. Завдяки цьому вони будуть “visible” в інших файлах.

var (
	// Список імен сплавів та мапа name → ID
	alloyNames []string
	alloyIDs   map[string]string

	// Для зберігання Entry-поле % для кожного інгредієнта сплаву
	alloyPercentageEntries map[string]map[string]*widget.Entry

	// Accordion, куди ми кладемо всі “Configure: <Alloy>” пункти
	percentageAccordion *widget.Accordion

	// VBox-контейнер, у якому будуть кольорові рядки деревовидного ASCII
	hierarchyContainer *fyne.Container

	// Таблиця підсумкових матеріалів (Material, mB, Ingots)
	summaryTable *widget.Table
	// Дані для цієї таблиці (рядки)
	summaryData [][]string

	// ID поточного вибраного сплаву (заповнюється після Select)
	currentAlloyID string

	// Поле вводу бажаної кількості (Entry)
	amountEntry *widget.Entry

	// RadioGroup для вибору “mB” чи “Ingots”
	modeRadio *widget.RadioGroup

	// Label для статусних повідомлень
	statusLabel *widget.Label
)
