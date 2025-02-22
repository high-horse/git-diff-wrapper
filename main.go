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

		// Get the file content before changes
		cmdOld := exec.Command("git", "show", "HEAD:"+filename)
		oldContent, err := cmdOld.Output()
		if err != nil {
			oldContent = []byte{}
		}

		// Get the current file content
		cmdNew := exec.Command("cat", filename)
		newContent, err := cmdNew.Output()
		if err != nil {
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

	// Sync horizontal scrolling between views
	leftView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyLeft, tcell.KeyRight:
			row, col := leftView.GetScrollOffset()
			if event.Key() == tcell.KeyLeft && col > 0 {
				col--
			} else if event.Key() == tcell.KeyRight {
				col++
			}
			leftView.ScrollTo(row, col)
			rightView.ScrollTo(row, col)
			app.Draw()
			return nil
		}
		return event
	})

	rightView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyLeft, tcell.KeyRight:
			row, col := rightView.GetScrollOffset()
			if event.Key() == tcell.KeyLeft && col > 0 {
				col--
			} else if event.Key() == tcell.KeyRight {
				col++
			}
			leftView.ScrollTo(row, col)
			rightView.ScrollTo(row, col)
			app.Draw()
			return nil
		}
		return event
	})

	// Sync vertical scrolling
	leftView.SetChangedFunc(func() {
		row, col := leftView.GetScrollOffset()
		rightView.ScrollTo(row, col)
		app.Draw()
	})

	rightView.SetChangedFunc(func() {
		row, col := rightView.GetScrollOffset()
		leftView.ScrollTo(row, col)
		app.Draw()
	})

	var fileNames []string
	for _, file := range files {
		fileNames = append(fileNames, file.filename)
	}
	dropdown.SetOptions(fileNames, func(text string, index int) {
		if index >= 0 && index < len(files) {
			displaySyncedDiff(files[index], leftView, rightView)
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

func parseFileWithDiff(filename, oldContent, newContent, diff string) DiffFile {
    oldLines := strings.Split(strings.TrimSpace(oldContent), "\n")
    newLines := strings.Split(strings.TrimSpace(newContent), "\n")
    
    // Handle empty files
    if len(oldLines) == 1 && oldLines[0] == "" {
        oldLines = []string{}
    }
    if len(newLines) == 1 && newLines[0] == "" {
        newLines = []string{}
    }

    // Parse diff to get changes
    hunks := parseHunks(diff)
    
    file := DiffFile{
        filename: filename,
        lines:    make([]DiffLine, 0),
    }

    oldIdx, newIdx := 0, 0
    
    // Process all hunks
    for _, hunk := range hunks {
        // Add unchanged lines before the hunk
        for oldIdx < hunk.oldStart-1 && oldIdx < len(oldLines) {
            file.lines = append(file.lines, DiffLine{
                leftNum:      oldIdx + 1,
                rightNum:     newIdx + 1,
                leftContent:  oldLines[oldIdx],
                rightContent: oldLines[oldIdx],
                state:       Normal,
            })
            oldIdx++
            newIdx++
        }

        // Process the changes in the hunk
        oldEnd := min(hunk.oldStart+hunk.oldCount, len(oldLines)+1)
        newEnd := min(hunk.newStart+hunk.newCount, len(newLines)+1)
        
        oldStart := max(0, hunk.oldStart-1)
        newStart := max(0, hunk.newStart-1)

        // Get the changed lines safely
        var oldHunkLines, newHunkLines []string
        if oldEnd > oldStart {
            oldHunkLines = oldLines[oldStart:oldEnd]
        }
        if newEnd > newStart {
            newHunkLines = newLines[newStart:newEnd]
        }

        // Create a mapping of modified lines
        modifiedPairs := findModifiedPairs(oldHunkLines, newHunkLines)

        tempOldIdx := oldStart
        tempNewIdx := newStart

        for tempOldIdx < oldEnd || tempNewIdx < newEnd {
            if pair, ok := modifiedPairs[tempOldIdx-oldStart]; ok && pair == tempNewIdx-newStart {
                // Modified line pair
                file.lines = append(file.lines, DiffLine{
                    leftNum:      tempOldIdx + 1,
                    rightNum:     tempNewIdx + 1,
                    leftContent:  oldLines[tempOldIdx],
                    rightContent: newLines[tempNewIdx],
                    state:       Modified,
                })
                tempOldIdx++
                tempNewIdx++
            } else if tempNewIdx < newEnd && !isInModifiedPairs(modifiedPairs, tempNewIdx-newStart) {
                // Added line
                file.lines = append(file.lines, DiffLine{
                    leftNum:      tempOldIdx + 1,
                    rightNum:     tempNewIdx + 1,
                    leftContent:  "",
                    rightContent: newLines[tempNewIdx],
                    state:       Added,
                })
                tempNewIdx++
            } else if tempOldIdx < oldEnd {
                // Removed line
                file.lines = append(file.lines, DiffLine{
                    leftNum:      tempOldIdx + 1,
                    rightNum:     tempNewIdx + 1,
                    leftContent:  oldLines[tempOldIdx],
                    rightContent: "",
                    state:       Removed,
                })
                tempOldIdx++
            } else {
                break
            }
        }

        oldIdx = oldEnd
        newIdx = newEnd
    }

    // Add remaining unchanged lines
    for oldIdx < len(oldLines) {
        file.lines = append(file.lines, DiffLine{
            leftNum:      oldIdx + 1,
            rightNum:     newIdx + 1,
            leftContent:  oldLines[oldIdx],
            rightContent: oldLines[oldIdx],
            state:       Normal,
        })
        oldIdx++
        newIdx++
    }

    return file
}

type Hunk struct {
	oldStart, oldCount int
	newStart, newCount int
}


// Helper functions for safe array bounds
func min(a, b int) int {
    if a < b {
        return a
    }
    return b
}

func max(a, b int) int {
    if a > b {
        return a
    }
    return b
}

func parseHunks(diff string) []Hunk {
    var hunks []Hunk
    lines := strings.Split(diff, "\n")
    
    for _, line := range lines {
        if strings.HasPrefix(line, "@@") {
            var hunk Hunk
            _, err := fmt.Sscanf(line, "@@ -%d,%d +%d,%d @@",
                &hunk.oldStart, &hunk.oldCount,
                &hunk.newStart, &hunk.newCount)
            if err == nil {
                hunks = append(hunks, hunk)
            }
        }
    }
    
    return hunks
}

func findModifiedPairs(oldLines, newLines []string) map[int]int {
	pairs := make(map[int]int)
	// Simple implementation - could be improved with diff matching algorithm
	for i, oldLine := range oldLines {
		for j, newLine := range newLines {
			if strings.TrimSpace(oldLine) == strings.TrimSpace(newLine) {
				pairs[i] = j
				break
			}
		}
	}
	return pairs
}

func isInModifiedPairs(pairs map[int]int, idx int) bool {
	for _, v := range pairs {
		if v == idx {
			return true
		}
	}
	return false
}

func displaySyncedDiff(file DiffFile, leftView, rightView *tview.TextView) {
	var leftContent, rightContent strings.Builder
	maxLineNumWidth := len(fmt.Sprint(len(file.lines)))
	lineNumFormat := fmt.Sprintf("%%%dd │ %%s\n", maxLineNumWidth)

	for _, line := range file.lines {
		switch line.state {
		case Normal:
			leftContent.WriteString(fmt.Sprintf(lineNumFormat, line.leftNum, line.leftContent))
			rightContent.WriteString(fmt.Sprintf(lineNumFormat, line.rightNum, line.rightContent))
		case Added:
			leftContent.WriteString(fmt.Sprintf(lineNumFormat, line.leftNum, ""))
			rightContent.WriteString("[green]" + fmt.Sprintf(lineNumFormat, line.rightNum, line.rightContent) + "[white]")
		case Removed:
			leftContent.WriteString("[red]" + fmt.Sprintf(lineNumFormat, line.leftNum, line.leftContent) + "[white]")
			rightContent.WriteString(fmt.Sprintf(lineNumFormat, line.rightNum, ""))
		case Modified:
			leftContent.WriteString("[yellow]" + fmt.Sprintf(lineNumFormat, line.leftNum, line.leftContent) + "[white]")
			rightContent.WriteString("[yellow]" + fmt.Sprintf(lineNumFormat, line.rightNum, line.rightContent) + "[white]")
		}
	}

	leftView.SetText(leftContent.String())
	rightView.SetText(rightContent.String())
}