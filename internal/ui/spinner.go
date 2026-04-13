package ui

import (
	"fmt"

	"github.com/charmbracelet/huh/spinner"
)

// Step represents a single operation in a multi-step workflow.
type Step struct {
	Title  string                 // e.g., "Creating Supabase project..."
	Action func() (string, error) // returns result text on success, or error
}

// RunSteps iterates through steps sequentially, showing a spinner for each one.
// On success it prints a green checkmark with the title and result. On failure
// it prints a red cross, the error message, and returns immediately.
func RunSteps(steps []Step) error {
	for _, step := range steps {
		var result string
		var actionErr error

		err := spinner.New().
			Title(step.Title).
			Action(func() {
				result, actionErr = step.Action()
			}).
			Run()
		if err != nil {
			fmt.Printf("%s %s\n", ErrorIcon, ErrorStyle.Render(step.Title))
			fmt.Printf("  %s\n", ErrorStyle.Render(err.Error()))
			return err
		}
		if actionErr != nil {
			fmt.Printf("%s %s\n", ErrorIcon, ErrorStyle.Render(step.Title))
			fmt.Printf("  %s\n", ErrorStyle.Render(actionErr.Error()))
			return actionErr
		}

		if result != "" {
			fmt.Printf("%s %s (%s)\n", SuccessIcon, SuccessStyle.Render(step.Title), MutedStyle.Render(result))
		} else {
			fmt.Printf("%s %s\n", SuccessIcon, SuccessStyle.Render(step.Title))
		}
	}
	return nil
}
