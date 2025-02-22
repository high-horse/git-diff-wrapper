# Git Diff TUI

A terminal-based Git diff viewer built with Go and `tview`. This tool provides a user-friendly interface to visualize `git diff` output, similar to VS Code's diff view, with color-coded changes, scrollable content, and a clean TUI layout.

![Git Diff TUI Screenshot](screenshot.png) <!-- Add a screenshot if available -->

---

## Features

- **Color-Coded Diffs**:
  - Removed lines are highlighted in red.
  - Added lines are highlighted in green.
  - File headers are highlighted in yellow.
  - Diff chunk headers are highlighted in purple.
  - File separators are highlighted in blue.

- **Scrollable Interface**:
  - Navigate through the diff output using arrow keys or `PgUp`/`PgDn`.

- **File Separators**:
  - Clear visual separators between different files in the diff.

- **Beautiful TUI Layout**:
  - The diff content is displayed inside a bordered box with a title.

---

## Installation

### Prerequisites

- Go (version 1.20 or higher)
- Git

### Steps

1. Clone the repository:
   ```bash
   git clone https://github.com/high-horse/git-diff-wrapper.git
   cd git-diff-tui
   ```
   
2. Install dependencies:
   ```bash   
   go get github.com/rivo/tview
   go get github.com/gdamore/tcell/v2
   ```

3. Build the project:
   ```bash   
   go build -o git-diff-tui
   ```

4. Run the executable:
   ```bash   
   ./git-diff-tui
   ```
