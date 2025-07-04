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
	lines    []DiffLine
}

type DiffLine struct {
	leftNum     int
	rightNum    int
	leftContent string
	rightContent string
	state       LineState
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

		// Get the diff for this file with full context
		cmdDiff := exec.Command("git", "diff", "--unified=999999", filename)
		diffOutput, err := cmdDiff.Output()
		if err != nil {
			fmt.Println("Error getting diff for", filename, ":", err)
			continue
		}

		file := parseGitDiff(filename, string(diffOutput))
		files = append(files, file)
	}

	app := tview.NewApplication()
	flex := tview.NewFlex().SetDirection(tview.FlexRow)

	dropdown := tview.NewDropDown().
		SetLabel("Select file: ").
		SetFieldWidth(50)

	leftView := tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(false).
		SetScrollable(true)
	rightView := tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(false).
		SetScrollable(true)

	// Create a channel to coordinate scroll synchronization
	scrollChan := make(chan struct{}, 1)

	// Function to synchronize scrolling between views
	syncScroll := func(source, target *tview.TextView) {
		select {
		case scrollChan <- struct{}{}:
			defer func() { <-scrollChan }()
			sx, sy := source.GetScrollOffset()
			tx, ty := target.GetScrollOffset()
			if sx != tx || sy != ty {
				target.ScrollTo(sx, sy)
				app.Draw()
			}
		default:
			// Skip if already syncing
		}
	}

	// Set up scroll synchronization
	leftView.SetChangedFunc(func() {
		syncScroll(leftView, rightView)
	})

	rightView.SetChangedFunc(func() {
		syncScroll(rightView, leftView)
	})

	// Handle arrow keys for scrolling
	leftView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyUp, tcell.KeyDown, tcell.KeyLeft, tcell.KeyRight:
			// Let the default behavior handle the scroll
			return event
		}
		return event
	})

	rightView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyUp, tcell.KeyDown, tcell.KeyLeft, tcell.KeyRight:
			// Let the default behavior handle the scroll
			return event
		}
		return event
	})

	var fileNames []string
	for _, file := range files {
		fileNames = append(fileNames, file.filename)
	}
	dropdown.SetOptions(fileNames, func(text string, index int) {
		if index >= 0 && index < len(files) {
			displaySyncedDiff(files[index], leftView, rightView)
			// Reset scroll position when changing files
			leftView.ScrollToBeginning()
			rightView.ScrollToBeginning()
		}
	})

	diffFlex := tview.NewFlex().
		AddItem(leftView, 0, 1, false).
		AddItem(rightView, 0, 1, false)

	flex.AddItem(dropdown, 1, 0, true).
		AddItem(diffFlex, 0, 1, false)

	leftView.SetBorder(true).SetTitle(" Original ")
	rightView.SetBorder(true).SetTitle(" Modified ")

	helpText := tview.NewTextView().
		SetDynamicColors(true).
		SetText("[yellow]Navigation: Arrow keys to scroll | Tab to switch focus | Ctrl-C to quit[white]").
		SetTextAlign(tview.AlignCenter)
	flex.AddItem(helpText, 1, 0, false)

	if len(files) > 0 {
		displaySyncedDiff(files[0], leftView, rightView)
		dropdown.SetCurrentOption(0)
	}

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

func parseGitDiff(filename, diff string) DiffFile {
	lines := strings.Split(diff, "\n")
	file := DiffFile{filename: filename}
	
	var oldLineNum, newLineNum int
	var hunkOldStart, hunkNewStart int
	
	for _, line := range lines {
		if strings.HasPrefix(line, "@@") {
			// Parse hunk header
			var oldCount, newCount int
			_, err := fmt.Sscanf(line, "@@ -%d,%d +%d,%d @@", 
				&hunkOldStart, &oldCount, 
				&hunkNewStart, &newCount)
			if err != nil {
				// Try without counts (for empty files)
				_, err = fmt.Sscanf(line, "@@ -%d +%d @@", 
					&hunkOldStart, &hunkNewStart)
				if err != nil {
					continue
				}
			}
			oldLineNum = hunkOldStart
			newLineNum = hunkNewStart
			continue
		}
		
		if len(line) == 0 {
			continue
		}
		
		prefix := line[0]
		content := line[1:]
		
		switch prefix {
		case ' ':
			// Unchanged line
			file.lines = append(file.lines, DiffLine{
				leftNum:     oldLineNum,
				rightNum:    newLineNum,
				leftContent: content,
				rightContent: content,
				state:      Normal,
			})
			oldLineNum++
			newLineNum++
		case '-':
			// Removed line
			file.lines = append(file.lines, DiffLine{
				leftNum:     oldLineNum,
				rightNum:    -1,
				leftContent: content,
				rightContent: "",
				state:      Removed,
			})
			oldLineNum++
		case '+':
			// Added line
			file.lines = append(file.lines, DiffLine{
				leftNum:     -1,
				rightNum:    newLineNum,
				leftContent: "",
				rightContent: content,
				state:      Added,
			})
			newLineNum++
		}
	}
	
	return file
}

func displaySyncedDiff(file DiffFile, leftView, rightView *tview.TextView) {
	var leftContent, rightContent strings.Builder
	
	// Find maximum line number for formatting
	maxLeftNum, maxRightNum := 0, 0
	for _, line := range file.lines {
		if line.leftNum > maxLeftNum {
			maxLeftNum = line.leftNum
		}
		if line.rightNum > maxRightNum {
			maxRightNum = line.rightNum
		}
	}
	
	leftNumWidth := len(fmt.Sprint(maxLeftNum))
	rightNumWidth := len(fmt.Sprint(maxRightNum))
	
	leftFormat := fmt.Sprintf("%%%dd │ %%s\n", leftNumWidth)
	rightFormat := fmt.Sprintf("%%%dd │ %%s\n", rightNumWidth)
	
	for _, line := range file.lines {
		// Left panel
		if line.leftNum == -1 {
			leftContent.WriteString(fmt.Sprintf("%*s │ \n", leftNumWidth, ""))
		} else {
			switch line.state {
			case Removed:
				leftContent.WriteString("[red]" + fmt.Sprintf(leftFormat, line.leftNum, line.leftContent) + "[white]")
			case Modified:
				leftContent.WriteString("[yellow]" + fmt.Sprintf(leftFormat, line.leftNum, line.leftContent) + "[white]")
			default:
				leftContent.WriteString(fmt.Sprintf(leftFormat, line.leftNum, line.leftContent))
			}
		}
		
		// Right panel
		if line.rightNum == -1 {
			rightContent.WriteString(fmt.Sprintf("%*s │ \n", rightNumWidth, ""))
		} else {
			switch line.state {
			case Added:
				rightContent.WriteString("[green]" + fmt.Sprintf(rightFormat, line.rightNum, line.rightContent) + "[white]")
			case Modified:
				rightContent.WriteString("[yellow]" + fmt.Sprintf(rightFormat, line.rightNum, line.rightContent) + "[white]")
			default:
				rightContent.WriteString(fmt.Sprintf(rightFormat, line.rightNum, line.rightContent))
			}
		}
	}
	
	leftView.SetText(leftContent.String())
	rightView.SetText(rightContent.String())
}