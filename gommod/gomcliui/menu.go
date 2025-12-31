package gomcliui

import (
	"fmt"

	"github.com/mikeschinkel/go-cliutil"
)

// MenuOption represents a single menu choice
type MenuOption struct {
	Label       string
	Description string
}

// MenuArgs contains arguments for displaying a menu
type MenuArgs struct {
	Prompt  string
	Options []MenuOption
	Writer  cliutil.Writer
}

// ShowMenu displays a vertical menu with bracket notation
// Returns:
//   - 0 to len(options)-1: selected option index
//   - -1: invalid selection
//   - -2: quit selected
func ShowMenu(args MenuArgs) (selectedIndex int, err error) {
	for {
		var choice rune

		// Display options with bracket notation
		args.Writer.Printf("\n")
		for i, opt := range args.Options {
			digit := i + 1
			args.Writer.Printf("[%d] %s\n", digit, opt.Label)
		}
		args.Writer.Printf("[0] help\n")
		args.Writer.Printf("[9] quit\n")

		// Display prompt
		args.Writer.Printf("\n%s ", args.Prompt)

		// Read single key
		choice, err = cliutil.ReadSingleKey()
		if err != nil {
			selectedIndex = -1
			goto end
		}

		// Echo the choice
		args.Writer.Printf("%c\n\n", choice)

		// Handle special keys
		switch choice {
		case '0':
			// Show help
			showHelp(args, args.Writer)
			continue // Redisplay menu

		case '9':
			// Quit
			selectedIndex = -2
			goto end

		case '1', '2', '3', '4', '5', '6', '7', '8':
			// Regular option selection
			selectedIndex = int(choice - '1')
			// Validate index is within range
			if selectedIndex >= len(args.Options) {
				args.Writer.Printf("Invalid option.\n")
				continue
			}
			goto end

		default:
			args.Writer.Printf("Invalid option.\n")
			continue
		}
	}

end:
	return selectedIndex, err
}

// ShowMenuInline displays a compact inline menu with bracket notation
// Format: [1] option1 [2] option2 — [0] help [9] quit
// Returns:
//   - 0 to len(options)-1: selected option index
//   - -1: invalid selection
//   - -2: quit selected
func ShowMenuInline(args MenuArgs) (selectedIndex int, err error) {
	for {
		var choice rune

		// Build option string with bracket notation
		var optStr string
		for i, opt := range args.Options {
			if i > 0 {
				optStr += " "
			}
			optStr += fmt.Sprintf("[%d] %s", i+1, opt.Label)
		}

		// Add separator and help/quit
		optStr += " — [0] help [9] quit"

		args.Writer.Printf("\n%s\n", optStr)
		args.Writer.Printf("%s ", args.Prompt)

		choice, err = cliutil.ReadSingleKey()
		if err != nil {
			selectedIndex = -1
			goto end
		}

		// Echo the choice
		args.Writer.Printf("%c\n\n", choice)

		// Handle special keys
		switch choice {
		case '0':
			// Show help
			showHelp(args, args.Writer)
			continue // Redisplay menu

		case '9':
			// Quit
			selectedIndex = -2
			goto end

		case '1', '2', '3', '4', '5', '6', '7', '8':
			// Regular option selection
			selectedIndex = int(choice - '1')
			// Validate index is within range
			if selectedIndex >= len(args.Options) {
				args.Writer.Printf("Invalid option.\n")
				continue
			}
			goto end

		default:
			args.Writer.Printf("Invalid option.\n")
			continue
		}
	}

end:
	return selectedIndex, err
}

// showHelp displays all menu options with their descriptions
func showHelp(args MenuArgs, writer cliutil.Writer) {
	writer.Printf("\nMenu Options:\n\n")
	for i, opt := range args.Options {
		if opt.Description != "" {
			writer.Printf("[%d] %s — %s\n", i+1, opt.Label, opt.Description)
		} else {
			writer.Printf("[%d] %s\n", i+1, opt.Label)
		}
	}
	writer.Printf("[0] help — Show this help message\n")
	writer.Printf("[9] quit — Exit this menu\n")
	writer.Printf("\n")
}
