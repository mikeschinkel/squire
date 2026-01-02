package gomtui

import (
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
