package squirescliui

import (
	"io"

	"github.com/mikeschinkel/go-dt/dtx"
)

// EditMessage opens the message in the user's editor for editing
func EditMessage(message string, writer io.Writer) (newMessage string, err error) {
	// TODO: Implement editor integration
	dtx.Fprintf(writer, "Editor not yet implemented.\n")
	newMessage = message
	return newMessage, nil
}
