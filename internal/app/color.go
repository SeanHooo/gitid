package app

import (
	"io"
	"os"
	"strings"

	"github.com/fatih/color"
)

// colorEnabled reports whether the output stream is a terminal and color has
// not been explicitly disabled via NO_COLOR. It is evaluated once, when the
// App is built, so piped or redirected output stays plain.
func colorEnabled(output io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	file, ok := output.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

// paint wraps s in the given color attributes, resetting afterward. It returns
// s unchanged when color is disabled so non-terminal output has no escape
// bytes. Coloring is delegated to fatih/color, which also enables Windows
// console VT processing where needed.
func (a App) paint(s string, attrs ...color.Attribute) string {
	if !a.Color {
		return s
	}
	c := color.New(attrs...)
	c.EnableColor()
	return c.Sprint(s)
}

func (a App) bold(s string) string   { return a.paint(s, color.Bold) }
func (a App) dim(s string) string    { return a.paint(s, color.Faint) }
func (a App) green(s string) string  { return a.paint(s, color.FgGreen) }
func (a App) yellow(s string) string { return a.paint(s, color.FgYellow) }
func (a App) cyan(s string) string   { return a.paint(s, color.FgCyan) }

// header renders a section heading.
func (a App) header(s string) string { return a.paint(s, color.Bold, color.FgCyan) }

// prog renders the program name.
func (a App) prog() string { return a.bold("gitid") }

// label renders a dimmed, right-padded field label so values line up.
func (a App) label(s string) string { return a.dim(padRight(s, 11)) }

// identity highlights the active profile name; the "(none)" placeholder is
// dimmed instead.
func (a App) identity(s string) string {
	if s == "" || s == "(none)" {
		return a.dim(s)
	}
	return a.paint(s, color.FgGreen, color.Bold)
}

// strong renders a value in bold, dimming known empty placeholders.
func (a App) strong(s string) string {
	if s == "" || s == "(none)" || s == "(unset)" {
		return a.dim(s)
	}
	return a.bold(s)
}

// ok renders the leading "OK:" marker for successful diagnostics.
func (a App) ok() string { return a.paint("OK:", color.FgGreen, color.Bold) }

// Red wraps s in red for terminal output, honoring NO_COLOR and TTY detection.
// It lets main print errors consistently with the rest of the CLI.
func Red(s string, output io.Writer) string {
	if !colorEnabled(output) {
		return s
	}
	c := color.New(color.FgRed)
	c.EnableColor()
	return c.Sprint(s)
}

// padRight pads s with trailing spaces to at least width visible columns. It
// must be applied before coloring so the padding width is computed from the
// plain text, not the escape codes.
func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
