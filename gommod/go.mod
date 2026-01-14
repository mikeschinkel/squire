module github.com/mikeschinkel/gomion/gommod

go 1.25.3

require (
	github.com/charmbracelet/bubbles v0.21.0
	github.com/charmbracelet/bubbletea v1.3.10
	github.com/charmbracelet/lipgloss v1.1.0
	github.com/evertras/bubble-table v0.19.2
	github.com/jedib0t/go-pretty/v6 v6.7.8
	github.com/lucasb-eyer/go-colorful v1.3.0
	github.com/mikeschinkel/go-cfgstore v0.4.1
	github.com/mikeschinkel/go-cliutil v0.3.0
	github.com/mikeschinkel/go-cliutil/climenu v0.0.0-00010101000000-000000000000
	github.com/mikeschinkel/go-dt v0.5.0
	github.com/mikeschinkel/go-dt/appinfo v0.2.1
	github.com/mikeschinkel/go-dt/dtx v0.2.1
	github.com/mikeschinkel/go-logutil v0.2.1
	go.dalton.dog/bubbleup v1.1.0
	golang.org/x/exp v0.0.0-20251219203646-944ab1f22d93
	golang.org/x/mod v0.31.0
	golang.org/x/term v0.38.0
	golang.org/x/tools v0.40.0
)

require (
	github.com/atotto/clipboard v0.1.4 // indirect
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/charmbracelet/colorprofile v0.3.3 // indirect
	github.com/charmbracelet/x/ansi v0.11.0 // indirect
	github.com/charmbracelet/x/cellbuf v0.0.14 // indirect
	github.com/charmbracelet/x/term v0.2.2 // indirect
	github.com/clipperhouse/displaywidth v0.4.1 // indirect
	github.com/clipperhouse/stringish v0.1.1 // indirect
	github.com/clipperhouse/uax29/v2 v2.3.0 // indirect
	github.com/erikgeiser/coninput v0.0.0-20211004153227-1c3628e74d0f // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-localereader v0.0.1 // indirect
	github.com/mattn/go-runewidth v0.0.19 // indirect
	github.com/muesli/ansi v0.0.0-20230316100256-276c6243b2f6 // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/muesli/reflow v0.3.0 // indirect
	github.com/muesli/termenv v0.16.0 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/text v0.32.0 // indirect
)

replace github.com/mikeschinkel/go-cfgstore => ../../go-pkgs/go-cfgstore
replace github.com/mikeschinkel/go-dt => ../../go-pkgs/go-dt

replace github.com/mikeschinkel/go-dt/dtx => ../../go-pkgs/go-dt/dtx

replace github.com/mikeschinkel/go-dt/appinfo => ../../go-pkgs/go-dt/appinfo

replace github.com/mikeschinkel/go-cliutil/climenu => ../../go-pkgs/go-cliutil/climenu

replace github.com/mikeschinkel/go-cliutil => ../../go-pkgs/go-cliutil

replace github.com/mikeschinkel/go-cfgstore => ../../go-pkgs/go-cfgstore

replace github.com/mikeschinkel/go-logutil => ../../go-pkgs/go-logutil

replace go.dalton.dog/bubbleup => ../../go-pkgs/bubbleup

replace github.com/evertras/bubble-table => ../../go-pkgs/bubble-table
