# TFC Alloy Calculator

This application calculates the required raw metal amounts needed to create specific alloys from the TerraFirmaCraft (TFC) mod for Minecraft. It provides a graphical user interface built with Go and the Fyne toolkit, and relies on a local MySQL database (launched via Docker Compose) to store alloy definitions and default percentages.

## Features

* **Calculate Raw Metal Requirements:** Computes exactly how many millibuckets (mB) or Ingots of each base metal (Copper, Zinc, Bismuth, Silver, Gold, Nickel, Pig Iron, etc.) are needed to produce your target alloy.
* **Dual Mode:** You can request your target amount either in mB or in Ingots, and the program will convert accordingly.
* **Configurable Percentages:** Expand the “Percentage Settings” accordion to override any ingredient percentages for the chosen alloy and its sub‐components—only within valid min/max ranges. If you do not customize, default (average) percentages are used.
* **Hierarchical Breakdown:** A colored, monospace ASCII‐tree on the right shows exactly how each intermediate component breaks down (with vertical bars and branch symbols in distinct colors by depth).
* **Final Summary Table:** Below the tree is a resizable table listing each base material’s total mB and Ingots required.
* **Cross-Platform GUI:** Built with the Fyne toolkit, it runs on Windows, macOS, and Linux (provided Go and a C compiler are installed).

## Prerequisites

Before building or running, ensure you have the following installed:

1. **Go:** Version 1.24 or later.
   ([Installation Guide](https://golang.org/doc/install))

2. **C Compiler:** A working C compiler (required by Fyne’s CGo code).

   * **Windows:** Install MinGW-w64 (e.g. via MSYS2 or from [https://mingw-w64.org](https://mingw-w64.org)). Make sure `gcc` is in your PATH.
   * **macOS:** Install Xcode Command Line Tools (`xcode-select --install`).
   * **Linux (Debian/Ubuntu):**

     ```sh
     sudo apt update
     sudo apt install build-essential libgl1-mesa-dev xorg-dev
     ```

3. **Docker & Docker Compose:** Used to run MySQL in a container.

   * [Get Docker](https://docs.docker.com/get-docker/)
   * [Get Docker Compose](https://docs.docker.com/compose/install/)

4. **Fyne Dependencies:** Follow the Fyne “Getting Started” guide if you encounter any missing libraries.
   ([Fyne Getting Started](https://developer.fyne.io/started/))

## Makefile Targets

A top‐level `Makefile` helps automate DB startup, tests, building, and running:

```makefile
# Shortcut to start DB (if needed), run unit tests, build the executable, and launch it:
make all

# Start MySQL container (if not running) and wait until it is accepting TCP connections:
make db-up

# Stop & remove the MySQL container:
make db-down

# Run all Go unit tests (calculator & data packages):
make test

# Build the Go binary (creates ./tfccalc):
make build

# Run the compiled binary (equivalent to ./tfccalc):
make run
```

Behind the scenes, `make db-up` checks `docker-compose ps -q mysql`, and if MySQL isn’t already running, it does `docker-compose up -d`, polls port 3306 until it is open, then sleeps a few extra seconds to allow initialization.

## Database Setup

1. **Project Root** contains a `docker-compose.yml` that defines a MySQL service named `mysql`.
2. When you run `make db-up`, Docker Compose will:

   * Create a default network `tfccalc_default`.
   * Launch a `tfccalc_mysql` container.
   * Wait until the container’s MySQL server is ready on `localhost:3306`.
3. The Go code (in `data/`) will connect to `root:password@tcp(127.0.0.1:3306)/tfccalc` to load alloy definitions and default percentages.

Whenever you alter the database schema or add new alloy entries, simply stop and restart:

```sh
make db-down
make db-up
```

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

3. **Ensure MySQL Is Running:**

   ```sh
   make db-up
   ```

   This will start the MySQL container (if not already running) and wait for it to be ready.

4. **Run Unit Tests:**

   ```sh
   make test
   ```

   (Calculator and data‐layer tests require the database to be up.)

5. **Build the Application:**

   ```sh
   make build
   ```

   Produces an executable named `tfccalc` (or `tfccalc.exe` on Windows).

6. **Launch the Application:**

   ```sh
   make run
   ```

   or, run the binary directly:

   ```sh
   ./tfccalc      # Linux/macOS
   tfccalc.exe    # Windows
   ```

   Alternatively, you can run via Go:

   ```sh
   go run main.go
   ```

## Usage

1. **Select Target Alloy:**
   In the dropdown, choose any alloy or “final steel” variant.

2. **Enter Desired Amount:**
   Type a positive number into the “Amount” field. This represents either mB or Ingots, depending on your selected mode.

3. **Select Mode (mB or Ingots):**
   Use the radio buttons to switch between millibuckets and ingots.

   * If you enter “10” in **mB** mode, it means 10 mB.
   * If you enter “10” in **Ingots** mode, it means 10 ingots (equal to 1000 mB).

4. **Configure Percentages (Optional):**
   Expand the “Percentage Settings” accordion on the left. You will see one or more items labeled:

   ```
   Configure: <AlloyName>
   ```

   Each section lists its ingredients and valid ranges (`[min–max%]`). You can type a custom percentage (e.g., “30.5”) for any ingredient to override the default breakdown. If you leave everything blank, default (average) percentages are applied.

   > **Important:** Each alloy’s ingredients must sum to 100%. The code enforces valid ranges. If you enter invalid or missing percentages, you’ll see validation errors.

5. **Click Calculate:**
   The right panel updates in two parts:

   * **Calculation Hierarchy (Top):**
     A scrollable, colored ASCII‐tree showing exactly how many mB of each intermediate alloy or raw material are required. Vertical bars (`│   `) and branch symbols (`├── `, `└── `) are colored by depth. Text is monospace.
   * **Final Summary (Bottom):**
     A resizable table listing each base metal (Copper, Zinc, Bismuth, etc.) with its required **mB** and **Ingots** totals.

6. **Resize as Needed:**
   You can drag the dividers between:

   * Left controls vs. right results
   * Within the right panel, between the hierarchy tree and the summary table
     to give more or less space to each section.

## Troubleshooting

* **MySQL Connection Errors:**
  If you see errors like

  ```
  Failed to initialize DB: cannot ping MySQL: invalid connection
  ```

  ensure that:

  1. Docker is running (`docker info`).
  2. MySQL container is up:

     ```sh
     docker-compose ps
     ```
  3. If needed, restart the container:

     ```sh
     make db-down
     make db-up
     ```
  4. Wait until the “MySQL is ready.” message appears.

* **Fyne Cannot Build / Linking Errors:**

  * Verify that `gcc` (or other C compiler) is installed and in your PATH.
  * On Windows, confirm you installed MinGW-w64 and added its `bin` folder to your PATH.
  * On Linux, install `build-essential`, `libgl1-mesa-dev`, `xorg-dev` if missing.

* **UI Text Overlapping or Too Small:**

  * You can drag the splitters between left/right panels and within the right panel to resize the tree area and the summary table.
  * If you resize the entire window to a very small size, the accordion labels may truncate. Expand the window to at least \~800×600.

* **Percentage Validation:**

  * Ensure that, for each alloy you configure, the sum of percentages across all ingredients equals 100%.
  * Each ingredient has a valid `[min–max%]` range—do not enter values outside that range.

---

Thank you for using **TFC Alloy Calculator**. If you encounter any bugs or have feature requests, please open an issue on the project’s GitHub repository.
