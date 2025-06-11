package root

import (
	"fmt"
	"go-diff/internal/ui"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var cached bool

var rootCmd = &cobra.Command{
	Use: "go-diff",
	Short: "View Git diff in terminal ui",
	Run: func(cmd *cobra.Command, args []string){
		m := ui.NewModel(cached)
		p := tea.NewProgram(m)
		
		if _, err := p.Run(); err != nil {
			fmt.Println("error : ", err)
			os.Exit(1)
		}
	},
}

func Execute() {
	rootCmd.Flags().BoolVarP(&cached, "ccched", "c", false, "Show staged diff (--cached)")
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}