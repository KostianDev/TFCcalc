# TFC Alloy Calculator

This application computes the required raw metal amounts needed to create specific alloys from the TerraFirmaCraft (TFC) mod for Minecraft, and provides an Anvil Forging Solver. It features a modern graphical user interface built with Go and the Fyne toolkit, and runs purely on local JSON data by default (no database required).

## Features

* **Calculate Raw Metal Requirements:** Computes exactly how many millibuckets (mB) or Ingots of each base metal (Copper, Zinc, Bismuth, Silver, Gold, Nickel, Pig Iron, etc.) are needed to produce your target alloy.
* **Dual Mode:** Request your target amount either in mB or in Ingots, and the program will convert accordingly.
* **Configurable Percentages:** Override any ingredient percentages for the chosen alloy and its sub‐components—only within valid min/max ranges. If not customized, default (average) percentages are used.
* **Hierarchical Breakdown:** A colored, monospace ASCII‐tree shows exactly how each intermediate component breaks down (with vertical bars and branch symbols in distinct colors by depth).
* **Final Summary Table:** A resizable table lists each base material’s total mB and Ingots required.
* **Anvil Forging Solver:** A dedicated tab with a slider for your target value (0-150), three visual "final action" slots, and an action palette to calculate the optimal anvil hit sequence (using an A* pathfinding algorithm).
* **Cross-Platform GUI:** Built with the Fyne toolkit, it runs seamlessly on Windows, macOS, and Linux.

## Prerequisites

Before building or running, ensure you have the following installed:

1. **Go:** Version 1.24 or later. ([Installation Guide](https://golang.org/doc/install))
2. **C Compiler:** A working C compiler (required by Fyne’s CGo code).
   * **Windows:** Install MinGW-w64. Make sure `gcc` is in your PATH.
   * **macOS:** Install Xcode Command Line Tools (`xcode-select --install`).
   * **Linux (Debian/Ubuntu):**
     ```sh
     sudo apt update
     sudo apt install build-essential libgl1-mesa-dev xorg-dev
     ```

*(Optional)* **Docker & Docker Compose:** Only needed if you explicitly want to run the MySQL-backed repository instead of the default JSON storage.

## Makefile Targets

A top‐level `Makefile` helps automate building, testing, and running:

```makefile
# Shortcut to run unit tests, build the executable, and launch it:
make all

# Run all Go unit tests:
make test

# Build the Go binary (creates ./tfccalc):
make build

# Run the compiled binary (equivalent to ./tfccalc):
make run

# --- Optional MySQL Targets ---
# Start MySQL container (if not running) and wait until it is accepting connections:
make db-up
# Stop & remove the MySQL container:
make db-down
```

## JSON Data (Default Storage)

Alloy data is stored locally in [assets/alloys.json](assets/alloys.json). The app loads it by default out of the box, requiring zero setup.

## Building and Running

Below is the typical workflow on any supported OS:

1. **Clone the Repository:**
   ```sh
   git clone <repository-url>
   cd <project-directory>
   ```

2. **Fetch Go Modules:**
   ```sh
   go mod tidy
   ```

3. **Run Unit Tests:**
   ```sh
   make test
   ```

4. **Build the Application:**
   ```sh
   make build
   ```
   *Produces an executable named `tfccalc` (or `tfccalc.exe` on Windows).*

5. **Launch the Application:**
   ```sh
   make run
   ```
   *Or run the binary directly: `./tfccalc` (Linux/macOS) or `tfccalc.exe` (Windows).*

## Usage

### Calculator Tab

1. **Select Target Alloy:** In the dropdown, choose any alloy or “final steel” variant.
2. **Enter Desired Amount:** Type a positive number into the “Amount” field.
3. **Select Mode (mB or Ingots):** Use the radio buttons to switch between millibuckets and ingots.
4. **Configure Percentages (Optional):** Expand the “Percentage Settings” accordion on the left. Type custom percentages to override the default breakdown.
5. **Click Calculate:** The right panel updates with your Calculation Hierarchy tree and Final Summary table.

### Anvil Tab

1. **Set Target Value:** Use the slider to select your required final target value (0-150).
2. **Select Final Actions:** 
   * Click on one of the three bottom slots (highlighted with an orange border).
   * Pick an action (Hit, Draw, Stamp, Bend, etc.) from the palette.
   * Repeat to set your required last three steps. (Leave as "Any" if undefined).
3. **Solve:** Click **Solve** to run the A* pathfinding algorithm. The exact forge sequence (along with step math values) will be displayed.
4. **Reset:** Use the Reset button to clear the board entirely.

## Database Setup (Optional/Secondary)

By default, the JSON file is utilized. If you wish to use the MySQL database:

1. Run `make db-up` to launch the Docker container.
2. Set the environment variable before running the app:
   ```sh
   export TFC_MYSQL_DSN="tfccalc_user:tfccalc_pass@tcp(127.0.0.1:3306)/tfccalc_db"
   ```
3. The Go code will detect the DSN and connect to MySQL instead.

---

Thank you for using **TFC Alloy Calculator**. If you encounter any bugs or have feature requests, please open an issue on the project’s GitHub repository.