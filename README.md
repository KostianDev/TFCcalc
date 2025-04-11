# TFC Alloy Calculator

This application calculates the required raw metal amounts needed to create specific alloys from the TerraFirmaCraft (TFC) mod for Minecraft. It provides a graphical user interface built with Go and the Fyne toolkit.

## Features

* Calculates base metal requirements (Copper, Zinc, Bismuth, Silver, Gold, Nickel, Pig Iron) for various TFC alloys.
* Supports calculation in both **mB** (millibuckets) and **Ingots**.
* Handles complex alloy dependencies, including the multi-step process for creating different types of Steel.
* Allows user configuration of ingredient percentages within their valid ranges for the target alloy and its sub-components.
* Displays a hierarchical breakdown of the required intermediate components in a tree view.
* Provides a final summary table listing the total required amount of each base metal.
* Cross-platform GUI using the Fyne toolkit.

## Prerequisites

Before building or running, ensure you have the following installed:

1.  **Go:** Version 1.24 or later. ([Installation Guide](https://golang.org/doc/install))
2.  **C Compiler:** A working C compiler (like GCC or Clang) is required by Fyne's dependencies (CGo).
    * **Windows:** MinGW-w64 (usually via MSYS2 or direct download) is recommended. Ensure GCC is in your system's PATH.
    * **macOS:** Xcode Command Line Tools.
    * **Linux:** `gcc` and relevant development packages (e.g., `build-essential`, `libgl1-mesa-dev`, `xorg-dev` on Debian/Ubuntu).
3.  **Fyne Dependencies:** Follow the Fyne "Getting Started" guide for any additional system-specific requirements. ([Fyne Getting Started](https://developer.fyne.io/started/))

## Building and Running

1.  **Clone the Repository:**
    ```sh
    git clone <repository-url>
    cd <project-directory> # e.g., cd TFCcalc
    ```
2.  **Fetch Dependencies:**
    ```sh
    go mod tidy
    ```
3.  **Build the Application:**
    ```sh
    go build .
    ```
    This will create an executable file (e.g., `tfccalc` or `tfccalc.exe`).
4.  **Run the Application:**
    * **From Executable:**
        ```sh
        # Linux/macOS
        ./tfccalc
        # Windows
        .\tfccalc.exe
        ```
    * **Directly with Go:**
        ```sh
        go run main.go
        ```

## Usage

1.  Launch the application.
2.  Select the **Target Alloy** you want to create from the dropdown menu.
3.  Enter the desired **Amount** (either mB or number of ingots).
4.  Select the calculation **Mode** ("mB" or "Ingots").
5.  Optionally, expand the **Percentage Settings** accordion sections for the main alloy and any sub-alloys you wish to customize. Enter specific percentages within the allowed ranges shown. Ensure the percentages for *each individual alloy* sum to 100%. If no custom percentages are entered, default (average) values will be used.
6.  Click the **Calculate** button.
7.  The results will be displayed on the right panel:
    * The **Calculation Breakdown** tree shows the hierarchy of required components.
    * The **Base Material Summary** table shows the total amount of each raw material needed.
