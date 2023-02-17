package xlog

import "fmt"

// Copied from zap/internal/color
// - adds coloring functionality for TTY output.

// Foreground colors.
const (
	colorBlack termColor = iota + 30
	colorRed
	colorGreen
	colorYellow
	colorBlue
	colorMagenta
	colorCyan
	colorWhite
)

func init() {
	// Disable linter unused-symbol-warnings
	_ = colorBlack
	_ = colorGreen
	_ = colorCyan
	_ = colorWhite
}

// termColor represents a text color.
type termColor uint8

// Add adds the coloring to the given string.
func (c termColor) Add(s string) string {
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", uint8(c), s)
}
