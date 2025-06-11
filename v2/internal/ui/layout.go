package ui

import (
	// "fmt"
	"strings"

	
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"go-diff/internal/git"
	"go-diff/internal/models"
	"go-diff/internal/parser"
)

var (
	borderStyle     = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Padding(0, 1)
	fileListStyle   = borderStyle.Copy().Width(30).BorderForeground(lipgloss.Color("8"))
	diffStyle       = borderStyle.Copy().BorderForeground(lipgloss.Color("7"))

	addStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	removeStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	headerStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
)


type model struct {
	list     list.Model
	diffData []models.DiffFile
	width    int
	height   int
}

func NewModel(cached bool) tea.Model {
	raw, err := git.GetDiff(cached)
	if err != nil {
		raw = "Error: " + err.Error()
	}

	diffFiles := parser.ParseGitDiff(raw)
	items := make([]list.Item, len(diffFiles))
	for i, file := range diffFiles {
		items[i] = listItem(file.FileName)
	}

	l := list.New(items, list.NewDefaultDelegate(), 50, 20)
	l.Title = "Changed Files"

	return model{
		list:     l,
		diffData: diffFiles,
		width:    100,
		height:   30,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}


func (m model) View() string {
	// Get selected file
	selected := m.list.SelectedItem()
	var diffContent string

	if selected != nil {
		for _, f := range m.diffData {
			if f.FileName == selected.FilterValue() {
				for _, h := range f.Hunks {
					diffContent += headerStyle.Render(h.Header) + "\n"
					for _, line := range h.Lines {
						switch line.Type {
						case "+":
							diffContent += addStyle.Render(line.Content) + "\n"
						case "-":
							diffContent += removeStyle.Render(line.Content) + "\n"
						default:
							diffContent += line.Content + "\n"
						}
					}
				}
				break
			}
		}
	}

	// Apply styles
	leftPane := fileListStyle.Render(m.list.View())
	rightPane := diffStyle.Render(diffContent)

	// Join panes horizontally
	return lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
}


// func (m model) View() string {
// 	// Determine file list width (30% of total width)
// 	listWidth := m.width / 3
// 	diffWidth := m.width - listWidth - 1

// 	// Render the file list
// 	m.list.SetSize(listWidth, m.height)
// 	left := m.list.View()

// 	// Get selected file's diff
// 	var right string
// 	selected := m.list.SelectedItem()
// 	if selected != nil {
// 		for _, f := range m.diffData {
// 			if f.FileName == selected.FilterValue() {
// 				for _, h := range f.Hunks {
// 					right += headerStyle.Render(h.Header) + "\n"
// 					for _, line := range h.Lines {
// 						switch line.Type {
// 						case "+":
// 							right += addStyle.Render(line.Content) + "\n"
// 						case "-":
// 							right += removeStyle.Render(line.Content) + "\n"
// 						default:
// 							right += line.Content + "\n"
// 						}
// 					}
// 				}
// 				break
// 			}
// 		}
// 	}

// 	// Split lines
// 	leftLines := strings.Split(left, "\n")
// 	rightLines := strings.Split(right, "\n")

// 	// Build side-by-side view
// 	var b strings.Builder
// 	maxLines := max(len(leftLines), len(rightLines))
// 	for i := 0; i < maxLines; i++ {
// 		var l, r string
// 		if i < len(leftLines) {
// 			l = padRight(leftLines[i], listWidth)
// 		} else {
// 			l = strings.Repeat(" ", listWidth)
// 		}

// 		if i < len(rightLines) {
// 			r = truncate(rightLines[i], diffWidth)
// 		}

// 		b.WriteString(fmt.Sprintf("%s│%s\n", l, r))
// 	}

// 	return b.String()
// }

func padRight(s string, w int) string {
	if len(s) > w {
		return s[:w]
	}
	return s + strings.Repeat(" ", w-len(s))
}

func truncate(s string, w int) string {
	if len(s) > w {
		return s[:w-1] + "…"
	}
	return s
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

type listItem string

func (i listItem) Title() string       { return string(i) }
func (i listItem) Description() string { return "" }
func (i listItem) FilterValue() string { return string(i) }
