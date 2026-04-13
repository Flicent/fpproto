package ui

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// ConfirmDestroy shows a destructive-action warning and prompts the user to
// type the prototype name to confirm. Returns true only if the typed name
// matches exactly.
func ConfirmDestroy(name string) (bool, error) {
	// Build the warning box.
	warningLines := []string{
		"",
		"  " + WarningStyle.Render("This will permanently destroy ") + AccentStyle.Render(name) + WarningStyle.Render(":"),
		"",
		"  " + ErrorStyle.Render("\u2022") + " Archive the GitHub repository",
		"  " + ErrorStyle.Render("\u2022") + " Delete the Supabase project",
		"  " + ErrorStyle.Render("\u2022") + " Delete the Vercel deployment",
		"",
	}

	warningBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Error).
		Padding(0, 1).
		Render(joinLines(warningLines))

	fmt.Println(warningBox)
	fmt.Println()

	var input string

	err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(WarningStyle.Render("Type \"" + name + "\" to confirm")).
				Value(&input).
				Validate(func(s string) error {
					if s != name {
						return fmt.Errorf("name does not match")
					}
					return nil
				}),
		),
	).Run()
	if err != nil {
		return false, err
	}

	return input == name, nil
}

// joinLines joins a slice of strings with newlines.
func joinLines(lines []string) string {
	result := ""
	for i, l := range lines {
		if i > 0 {
			result += "\n"
		}
		result += l
	}
	return result
}
