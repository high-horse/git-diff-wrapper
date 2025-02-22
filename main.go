package main

import (
	"fmt"
	"os/exec"
	"strings"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type DiffFile struct {
	filename    string
	leftLines   []Line
	rightLines  []Line
}

type Line struct {
	content string
	state   LineState // Normal, Added, Removed, or Modified
}

type LineState int

const (
	Normal LineState = iota
	Added
	Removed
	Modified
)

func main() {
	// Run git diff to get the list of changed files
	cmd := exec.Command("git", "diff", "--name-only")
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("Error running git diff:", err)
		return
	}

	changedFiles := strings.Split(strings.TrimSpace(string(output)), "\n")
	var files []DiffFile

	for _, filename := range changedFiles {
		if filename == "" {
			continue
		}

		// Get the file content before changes
		cmdOld := exec.Command("git", "show", "HEAD:"+filename)
		oldContent, err := cmdOld.Output()
		if err != nil {
			// File might be new
			oldContent = []byte{}
		}

		// Get the current file content
		cmdNew := exec.Command("cat", filename)
		newContent, err := cmdNew.Output()
		if err != nil {
			// File might be deleted
			newContent = []byte{}
		}

		// Get the diff for this file
		cmdDiff := exec.Command("git", "diff", "--unified=0", filename)
		diffOutput, err := cmdDiff.Output()
		if err != nil {
			fmt.Println("Error getting diff for", filename, ":", err)
			continue
		}

		file := parseFileWithDiff(filename, string(oldContent), string(newContent), string(diffOutput))
		files = append(files, file)
	}

	// Create the TUI application
	app := tview.NewApplication()

	// Create the main flex container
	flex := tview.NewFlex().SetDirection(tview.FlexRow)

	// Create a dropdown to select files
	dropdown := tview.NewDropDown().
		SetLabel("Select file: ").
		SetFieldWidth(50)

	// Create text views for left and right panels
	leftView := tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(false).
		SetScrollable(true)
	rightView := tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(false).
		SetScrollable(true)

	// Add horizontal scrolling handlers
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

	// Sync vertical scrolling between views
	leftView.SetChangedFunc(func() {
		row, _ := leftView.GetScrollOffset()
		_, col := rightView.GetScrollOffset()
		rightView.ScrollTo(row, col)
		app.Draw()
	})

	rightView.SetChangedFunc(func() {
		row, _ := rightView.GetScrollOffset()
		_, col := leftView.GetScrollOffset()
		leftView.ScrollTo(row, col)
		app.Draw()
	})

	// Add file names to dropdown
	var fileNames []string
	for _, file := range files {
		fileNames = append(fileNames, file.filename)
	}
	dropdown.SetOptions(fileNames, func(text string, index int) {
		if index >= 0 && index < len(files) {
			displayFullDiff(files[index], leftView, rightView)
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

	// Add help text
	helpText := tview.NewTextView().
		SetDynamicColors(true).
		SetText("[yellow]Navigation: Arrow keys to scroll | Tab to switch focus | Ctrl-C to quit[white]").
		SetTextAlign(tview.AlignCenter)
	flex.AddItem(helpText, 1, 0, false)

	// Show first file if available
	if len(files) > 0 {
		displayFullDiff(files[0], leftView, rightView)
		dropdown.SetCurrentOption(0)
	}

	// Enable focus switching
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			current := app.GetFocus()
			switch current {
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

	if err := app.SetRoot(flex, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}

func parseFileWithDiff(filename, oldContent, newContent, diff string) DiffFile {
	// Parse the diff to get the changed line numbers
	changes := make(map[int]LineState) // line number -> state
	lines := strings.Split(diff, "\n")
	var currentOldLine, currentNewLine int

	for _, line := range lines {
		if strings.HasPrefix(line, "@@") {
			// Parse the hunk header
			var oldStart, oldCount, newStart, newCount int
			fmt.Sscanf(line, "@@ -%d,%d +%d,%d @@", &oldStart, &oldCount, &newStart, &newCount)
			currentOldLine = oldStart
			currentNewLine = newStart
			continue
		}

		if strings.HasPrefix(line, "-") {
			changes[currentOldLine] = Removed
			currentOldLine++
		} else if strings.HasPrefix(line, "+") {
			changes[currentNewLine] = Added
			currentNewLine++
		} else if !strings.HasPrefix(line, "diff") && !strings.HasPrefix(line, "index") {
			currentOldLine++
			currentNewLine++
		}
	}

	// Create the file structure with both versions
	file := DiffFile{
		filename: filename,
	}

	// Process old content
	oldLines := strings.Split(oldContent, "\n")
	for i, line := range oldLines {
		lineNum := i + 1
		state := Normal
		if s, exists := changes[lineNum]; exists {
			state = s
		}
		file.leftLines = append(file.leftLines, Line{
			content: line,
			state:   state,
		})
	}

	// Process new content
	newLines := strings.Split(newContent, "\n")
	for i, line := range newLines {
		lineNum := i + 1
		state := Normal
		if s, exists := changes[lineNum]; exists {
			state = s
		}
		file.rightLines = append(file.rightLines, Line{
			content: line,
			state:   state,
		})
	}

	return file
}

func displayFullDiff(file DiffFile, leftView, rightView *tview.TextView) {
	var leftContent, rightContent strings.Builder

	// Add line numbers and content for left view
	for i, line := range file.leftLines {
		lineNum := fmt.Sprintf("%4d | ", i+1)
		switch line.state {
		case Removed:
			leftContent.WriteString("[red]" + lineNum + line.content + "[white]\n")
		case Modified:
			leftContent.WriteString("[yellow]" + lineNum + line.content + "[white]\n")
		default:
			leftContent.WriteString(lineNum + line.content + "\n")
		}
	}

	// Add line numbers and content for right view
	for i, line := range file.rightLines {
		lineNum := fmt.Sprintf("%4d | ", i+1)
		switch line.state {
		case Added:
			rightContent.WriteString("[green]" + lineNum + line.content + "[white]\n")
		case Modified:
			rightContent.WriteString("[yellow]" + lineNum + line.content + "[white]\n")
		default:
			rightContent.WriteString(lineNum + line.content + "\n")
		}
	}

	leftView.SetText(leftContent.String())
	rightView.SetText(rightContent.String())
}