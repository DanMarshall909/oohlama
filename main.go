package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"please/config"
	"please/models"
	"please/providers"
	"please/types"
	"please/ui"
)

func main() {
	// Check if we're being run as "pls" or "ol" (legacy) with special flags
	programName := filepath.Base(os.Args[0])
	if programName == "pls" || programName == "pls.exe" || programName == "ol" || programName == "ol.exe" {
		// Running as the short alias
	}

	// Handle special commands
	if len(os.Args) >= 2 {
		switch os.Args[1] {
		case "--install-alias":
			installAlias()
			return
		case "--uninstall-alias":
			uninstallAlias()
			return
		case "--version":
			ui.ShowVersion()
			return
		case "--help", "-h":
			ui.ShowHelp()
			return
		}
	}

	// If no arguments provided, show interactive main menu
	if len(os.Args) < 2 {
		ui.ShowMainMenu()
		return
	}

	// Join all arguments after the program name as the task description
	taskDescription := strings.Join(os.Args[1:], " ")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		// Create default config if none exists
		cfg = config.CreateDefault()
		config.Save(cfg) // Ignore errors for config saving
	}

	// Determine script type and provider
	scriptType := config.DetermineScriptType(cfg)
	provider := config.DetermineProvider(cfg)

	// Select the best model for the task
	model, err := models.SelectBestModel(cfg, taskDescription, provider)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not auto-select model (%v), using fallback\n", err)
		// Use fallback based on provider
		model = getFallbackModel(provider)
	}

	// Create the script request
	request := &types.ScriptRequest{
		TaskDescription: taskDescription,
		ScriptType:      scriptType,
		Provider:        provider,
		Model:           model,
	}

	// Generate script using the appropriate provider
	response, err := generateScript(cfg, request)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Display the script with explanation and ask for confirmation
	displayScriptAndConfirm(response)
}

// generateScript creates a script using the appropriate provider
func generateScript(cfg *types.Config, request *types.ScriptRequest) (*types.ScriptResponse, error) {
	var provider providers.Provider

	switch request.Provider {
	case "ollama":
		provider = providers.NewOllamaProvider(cfg)
	case "openai":
		provider = providers.NewOpenAIProvider(cfg)
	case "anthropic":
		provider = providers.NewAnthropicProvider(cfg)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", request.Provider)
	}

	if !provider.IsConfigured(cfg) {
		return nil, fmt.Errorf("provider %s is not properly configured", request.Provider)
	}

	return provider.GenerateScript(request)
}

// getFallbackModel returns a fallback model based on provider
func getFallbackModel(provider string) string {
	switch provider {
	case "openai":
		return "gpt-3.5-turbo"
	case "anthropic":
		return "claude-3-haiku-20240307"
	default:
		return "llama3.2"
	}
}

// displayScriptAndConfirm shows the generated script with explanation and interactive menu
func displayScriptAndConfirm(response *types.ScriptResponse) {
	fmt.Printf("╔══════════════════════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║                           🤖 Please Script Generator                         ║\n")
	fmt.Printf("╚══════════════════════════════════════════════════════════════════════════════╝\n\n")

	fmt.Printf("📝 Task: %s\n", response.TaskDescription)
	fmt.Printf("🧠 Model: %s (%s)\n", response.Model, response.Provider)
	fmt.Printf("🖥️  Platform: %s script\n", response.ScriptType)

	fmt.Printf("\n╔══════════════════════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║                              📋 Generated Script                             ║\n")
	fmt.Printf("╚══════════════════════════════════════════════════════════════════════════════╝\n\n")

	// Display the script with line numbers
	lines := strings.Split(response.Script, "\n")
	for i, line := range lines {
		lineNum := fmt.Sprintf("%3d", i+1)
		fmt.Printf("\033[90m%s│\033[0m %s\n", lineNum, line)
	}

	fmt.Printf("\n✅ Script generated successfully!\n")
	
	// Show interactive menu
	ui.ShowScriptMenu(response)
}

// installAlias creates the "pls" shortcut for the current platform
func installAlias() {
	ui.PrintRainbowBanner()
	fmt.Printf("\n%s🔧 Installing 'pls' alias (with 'ol' for backwards compatibility)...%s\n\n", ui.ColorBold+ui.ColorYellow, ui.ColorReset)
	
	// Get current executable path
	execPath, err := os.Executable()
	if err != nil {
		fmt.Printf("%s❌ Failed to get executable path: %v%s\n", ui.ColorRed, err, ui.ColorReset)
		return
	}

	dir := filepath.Dir(execPath)
	batContent := fmt.Sprintf(`@echo off
"%s" %%*
`, execPath)

	// Create pls.bat as the primary alias
	plsBatPath := filepath.Join(dir, "pls.bat")
	if err := os.WriteFile(plsBatPath, []byte(batContent), 0755); err != nil {
		fmt.Printf("%s❌ Failed to create pls.bat: %v%s\n", ui.ColorRed, err, ui.ColorReset)
		return
	}

	// Create ol.bat for backwards compatibility
	olBatPath := filepath.Join(dir, "ol.bat")
	if err := os.WriteFile(olBatPath, []byte(batContent), 0755); err != nil {
		fmt.Printf("%s⚠️ Warning: Failed to create ol.bat for backwards compatibility: %v%s\n", ui.ColorYellow, err, ui.ColorReset)
	}

	fmt.Printf("%s✅ Successfully created pls.bat!%s\n", ui.ColorGreen, ui.ColorReset)
	fmt.Printf("%s✅ Created ol.bat for backwards compatibility%s\n\n", ui.ColorGreen, ui.ColorReset)
	ui.PrintInstallationSuccess()
}

// uninstallAlias removes both "pls" and "ol" shortcuts
func uninstallAlias() {
	ui.PrintRainbowBanner()
	fmt.Printf("\n%s🗑️  Removing aliases...%s\n\n", ui.ColorBold+ui.ColorYellow, ui.ColorReset)

	// Look for aliases in the same directory as the executable
	execPath, err := os.Executable()
	if err != nil {
		fmt.Printf("%s❌ Failed to get executable path: %v%s\n", ui.ColorRed, err, ui.ColorReset)
		return
	}

	dir := filepath.Dir(execPath)
	
	// Remove pls.bat
	plsBatPath := filepath.Join(dir, "pls.bat")
	if err := os.Remove(plsBatPath); err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("%s💭 pls.bat not found%s\n", ui.ColorYellow, ui.ColorReset)
		} else {
			fmt.Printf("%s❌ Failed to remove pls.bat: %v%s\n", ui.ColorRed, err, ui.ColorReset)
		}
	} else {
		fmt.Printf("%s✅ Successfully removed pls.bat%s\n", ui.ColorGreen, ui.ColorReset)
	}

	// Remove ol.bat
	olBatPath := filepath.Join(dir, "ol.bat")
	if err := os.Remove(olBatPath); err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("%s💭 ol.bat not found%s\n", ui.ColorYellow, ui.ColorReset)
		} else {
			fmt.Printf("%s❌ Failed to remove ol.bat: %v%s\n", ui.ColorRed, err, ui.ColorReset)
		}
	} else {
		fmt.Printf("%s✅ Successfully removed ol.bat%s\n", ui.ColorGreen, ui.ColorReset)
	}
}
