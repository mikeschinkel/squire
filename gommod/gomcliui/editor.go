package gomcliui

import (
	"github.com/mikeschinkel/go-cliutil"
)

// EditMessage opens the message in the user's editor for editing
func EditMessage(message string, writer cliutil.Writer) (newMessage string, err error) {
	// TODO: Implement editor integration
	writer.Printf("Editor not yet implemented.\n")
	return message, err
}
