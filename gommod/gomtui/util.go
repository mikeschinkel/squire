package gomtui

import (
	"fmt"
	"log/slog"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var lipglossStyle = lipgloss.NewStyle()

func reuseLipglossStyle() lipgloss.Style {
	return lipglossStyle
}
func styleWithRGBColor(color RGBColor) lipgloss.Style {
	return lipglossStyle.Foreground(lipgloss.Color(color))
}
func renderRGBColor[T ~string](content T, color RGBColor) string {
	return styleWithRGBColor(color).Render(string(content))
}

func teaMsgAttrs(msg tea.Msg) (attr slog.Attr) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		attr = slog.String("msg_type", fmt.Sprintf("%T", msg))
		goto end
	}
	attr = slog.String("key_msg", km.String())
end:
	return attr
}

func appendCmd(cmds []tea.Cmd, cmd tea.Cmd) []tea.Cmd {
	if cmd == nil {
		goto end
	}
	if len(cmds) == 0 {
		cmds = []tea.Cmd{cmd}
		goto end
	}
	cmds = append(cmds, cmd)
end:
	return cmds
}
