// Package graphql provides GraphQL schema exploration capabilities
package graphql

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/xaaha/hulak/pkg/envparser"
	"github.com/xaaha/hulak/pkg/utils"
)

// ExplorerConfig holds configuration for the GraphQL explorer
type ExplorerConfig struct {
	EnvFiles    []string
	SelectedEnv string
	ProjectRoot string
}

// StartExplorer initializes and starts the GraphQL explorer TUI
func StartExplorer() error {
	// Check if project is initialized
	if !isProjectInitialized() {
		utils.PrintRed("Project not initialized. Please run 'hulak init' first.")
		return utils.ColorError("project not initialized")
	}

	// Get available environment files
	envFiles, err := utils.GetEnvFiles()
	if err != nil {
		return utils.ColorError("failed to get environment files: %w", err)
	}

	if len(envFiles) == 0 {
		utils.PrintWarning("No environment files found. Run 'hulak init' to create them.")
		return utils.ColorError("no environment files found")
	}

	config := ExplorerConfig{
		EnvFiles:    envFiles,
		SelectedEnv: utils.DefaultEnvVal,
	}

	// Initialize the TUI model
	m := initialModel(config)
	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return utils.ColorError("error running TUI: %w", err)
	}

	return nil
}

// isProjectInitialized checks if the hulak project has been initialized
func isProjectInitialized() bool {
	envPath, err := utils.CreatePath(utils.EnvironmentFolder)
	if err != nil {
		return false
	}

	// Check if env directory exists
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		return false
	}

	// Check if global.env exists
	globalEnvPath := envPath + "/" + utils.DefaultEnvVal + utils.DefaultEnvFileSuffix
	if _, err := os.Stat(globalEnvPath); os.IsNotExist(err) {
		return false
	}

	return true
}

// LoadEnvironment loads the selected environment variables
func LoadEnvironment(envName string) (map[string]any, error) {
	secretsMap, err := envparser.GenerateSecretsMap(envName)
	if err != nil {
		return nil, utils.ColorError("failed to load environment: %w", err)
	}
	return secretsMap, nil
}
