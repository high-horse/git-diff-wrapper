package main

import (
	"fmt"
	"os/exec"
	"strings"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type DiffFile struct {
	filename string
	hunks    []DiffHunk
}

type DiffHunk struct {
	header  string
	left    []string
	right   []string
}

func main() {
	// Run git diff and capture the output
	cmd := exec.Command("git", "diff")
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("Error running git diff:", err)
		return
	}

	// Parse the git diff output
	files := parseGitDiff(string(output))

	// Create the TUI application
	app := tview.NewApplication()

	// Create the main flex container
	flex := tview.NewFlex().SetDirection(tview.FlexRow)

	// Create a dropdown to select files
	dropdown := tview.NewDropDown().
		SetLabel("Select file: ").
		SetFieldWidth(50)

	// Create text views for left and right panels with horizontal scrolling
	leftView := tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(false).  // Disable wrapping to enable horizontal scrolling
		SetScrollable(true)
	rightView := tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(false).  // Disable wrapping to enable horizontal scrolling
		SetScrollable(true)

	// Add keyboard navigation for horizontal scrolling
	leftView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyLeft:
			row, col := leftView.GetScrollOffset()
			if col > 0 {
				leftView.ScrollTo(row, col-1)
			}
		case tcell.KeyRight:
			row, col := leftView.GetScrollOffset()
			leftView.ScrollTo(row, col+1)
		}
		return event
	})

	rightView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyLeft:
			row, col := rightView.GetScrollOffset()
			if col > 0 {
				rightView.ScrollTo(row, col-1)
			}
		case tcell.KeyRight:
			row, col := rightView.GetScrollOffset()
			rightView.ScrollTo(row, col+1)
		}
		return event
	})

	// Add file names to dropdown
	var fileNames []string
	for _, file := range files {
		fileNames = append(fileNames, file.filename)
	}
	dropdown.SetOptions(fileNames, func(text string, index int) {
		// When a file is selected, update both views
		if index >= 0 && index < len(files) {
			displayDiff(files[index], leftView, rightView)
		}
	})

	// Create a flex container for the diff views
	diffFlex := tview.NewFlex().
		AddItem(leftView, 0, 1, false).
		AddItem(rightView, 0, 1, false)

	// Add the dropdown and diff views to the main flex
	flex.AddItem(dropdown, 1, 0, true).
		AddItem(diffFlex, 0, 1, false)

	// Set up borders and titles
	leftView.SetBorder(true).SetTitle(" Original ")
	rightView.SetBorder(true).SetTitle(" Modified ")

	// Add key bindings help text
	helpText := tview.NewTextView().
		SetDynamicColors(true).
		SetText("[yellow]Navigation: Arrow keys to scroll | Tab to switch focus | Ctrl-C to quit[white]").
		SetTextAlign(tview.AlignCenter)
	flex.AddItem(helpText, 1, 0, false)

	// If there are files, display the first one
	if len(files) > 0 {
		displayDiff(files[0], leftView, rightView)
		dropdown.SetCurrentOption(0)
	}

	// Enable switching focus between views
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyTab:
			// Cycle focus between dropdown, left view, and right view
			currentFocus := app.GetFocus()
			switch currentFocus {
			case dropdown:
				app.SetFocus(leftView)
			case leftView:
				app.SetFocus(rightView)
			case rightView:
				app.SetFocus(dropdown)
			}
			return nil
		}
		return event
	})

	// Run the application
	if err := app.SetRoot(flex, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}

func displayDiff(file DiffFile, leftView, rightView *tview.TextView) {
	var leftContent, rightContent strings.Builder

	for _, hunk := range file.hunks {
		leftContent.WriteString("[purple]" + hunk.header + "[white]\n")
		rightContent.WriteString("[purple]" + hunk.header + "[white]\n")

		// Add left content
		for _, line := range hunk.left {
			if strings.HasPrefix(line, "-") {
				leftContent.WriteString("[red]" + strings.TrimPrefix(line, "-") + "[white]\n")
			} else {
				leftContent.WriteString(line + "\n")
			}
		}

		// Add right content
		for _, line := range hunk.right {
			if strings.HasPrefix(line, "+") {
				rightContent.WriteString("[green]" + strings.TrimPrefix(line, "+") + "[white]\n")
			} else {
				rightContent.WriteString(line + "\n")
			}
		}
	}

	leftView.SetText(leftContent.String())
	rightView.SetText(rightContent.String())
}

func parseGitDiff(diff string) []DiffFile {
	var files []DiffFile
	var currentFile *DiffFile
	var currentHunk *DiffHunk
	lines := strings.Split(diff, "\n")

	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "diff --git"):
			// Start a new file
			if currentFile != nil {
				if currentHunk != nil {
					currentFile.hunks = append(currentFile.hunks, *currentHunk)
				}
				files = append(files, *currentFile)
			}
			currentFile = &DiffFile{
				filename: strings.TrimPrefix(line, "diff --git "),
			}
			currentHunk = nil

		case strings.HasPrefix(line, "@@"):
			// Start a new hunk
			if currentHunk != nil && currentFile != nil {
				currentFile.hunks = append(currentFile.hunks, *currentHunk)
			}
			currentHunk = &DiffHunk{
				header: line,
			}

		case strings.HasPrefix(line, "-"):
			if currentHunk != nil {
				currentHunk.left = append(currentHunk.left, line)
			}

		case strings.HasPrefix(line, "+"):
			if currentHunk != nil {
				currentHunk.right = append(currentHunk.right, line)
			}

		case strings.HasPrefix(line, " "):
			if currentHunk != nil {
				currentHunk.left = append(currentHunk.left, line)
				currentHunk.right = append(currentHunk.right, line)
			}
		}
	}

	// Add the last file and hunk
	if currentFile != nil {
		if currentHunk != nil {
			currentFile.hunks = append(currentFile.hunks, *currentHunk)
		}
		files = append(files, *currentFile)
	}

	return files
}