// Package ui provides terminal output formatting, colors, spinners, and structured
// error display for the OneConfig CLI.
package ui

import (
	"fmt"
	"os"

	"github.com/pterm/pterm"
)

func init() {
	// Disable pterm's default header to prevent blank lines in output
	pterm.DisableDebugMessages()
}

var (
	// Brand colors
	PrimaryColor = pterm.NewStyle(pterm.FgCyan, pterm.Bold)
	SuccessColor = pterm.NewStyle(pterm.FgGreen)
	WarningColor = pterm.NewStyle(pterm.FgYellow)
	ErrorColor   = pterm.NewStyle(pterm.FgRed, pterm.Bold)
	MutedColor   = pterm.NewStyle(pterm.FgGray)
	BoldStyle    = pterm.NewStyle(pterm.Bold)
)

// Banner prints the OneConfig styled banner.
func Banner() {
	cyan := pterm.NewStyle(pterm.FgCyan, pterm.Bold)
	lightCyan := pterm.NewStyle(pterm.FgLightCyan, pterm.Bold)
	fmt.Println()
	cyan.Print("  ⚡ One")
	lightCyan.Println("Config")
	MutedColor.Println("  Set up any dev environment with one command")
	fmt.Println()
}

// Header prints a section header.
func Header(text string) {
	fmt.Println()
	PrimaryColor.Println("▸ " + text)
	fmt.Println()
}

// Success prints a success message.
func Success(text string) {
	SuccessColor.Println("  ✓ " + text)
}

// Warning prints a warning message.
func Warning(text string) {
	WarningColor.Println("  ⚠ " + text)
}

// Error prints an error message with optional hint.
func Error(msg string, hint string) {
	ErrorColor.Println("  ✗ " + msg)
	if hint != "" {
		fmt.Println()
		MutedColor.Println("    💡 " + hint)
	}
}

// Fatal prints an error and exits with code 1.
func Fatal(msg string, hint string) {
	Error(msg, hint)
	os.Exit(1)
}

// Info prints an informational message.
func Info(text string) {
	MutedColor.Println("  ℹ " + text)
}

// Step prints a step description with a bullet.
func Step(text string) {
	fmt.Println("  • " + text)
}

// Spinner creates and returns a new spinner.
func Spinner(text string) (*pterm.SpinnerPrinter, error) {
	return pterm.DefaultSpinner.
		WithRemoveWhenDone(true).
		WithText(text).
		Start()
}

// Table renders a simple table.
func Table(headers []string, rows [][]string) {
	data := pterm.TableData{headers}
	for _, row := range rows {
		data = append(data, row)
	}
	pterm.DefaultTable.WithHasHeader().WithData(data).Render()
}

// StatusBadge returns a colored status string.
func StatusBadge(status string) string {
	switch status {
	case "running", "healthy", "ready":
		return SuccessColor.Sprint("● " + status)
	case "starting", "pending":
		return WarningColor.Sprint("◐ " + status)
	case "stopped", "error", "failed":
		return ErrorColor.Sprint("○ " + status)
	default:
		return MutedColor.Sprint("? " + status)
	}
}
