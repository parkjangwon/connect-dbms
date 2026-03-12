package cmd

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"oslo/internal/profile"
	"oslo/internal/tui"
)

func runTUI(cmd *cobra.Command, args []string) error {
	store, err := profile.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	app := tui.NewApp(store)
	p := tea.NewProgram(app, tea.WithAltScreen())

	_, err = p.Run()
	return err
}
