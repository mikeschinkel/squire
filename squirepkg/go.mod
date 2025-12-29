module github.com/mikeschinkel/squire/squirepkg

go 1.25.3

require (
	github.com/jedib0t/go-pretty/v6 v6.7.8
	github.com/mikeschinkel/go-cfgstore v0.4.1
	github.com/mikeschinkel/go-cliutil v0.3.0
	github.com/mikeschinkel/go-cliutil/climenu v0.0.0-00010101000000-000000000000
	github.com/mikeschinkel/go-dt v0.5.0
	github.com/mikeschinkel/go-dt/appinfo v0.2.1
	github.com/mikeschinkel/go-dt/dtx v0.2.1
	github.com/mikeschinkel/go-logutil v0.2.1
	golang.org/x/exp v0.0.0-20251219203646-944ab1f22d93
	golang.org/x/mod v0.31.0
	golang.org/x/tools v0.40.0
)

require (
	github.com/clipperhouse/stringish v0.1.1 // indirect
	github.com/clipperhouse/uax29/v2 v2.3.0 // indirect
	github.com/mattn/go-runewidth v0.0.19 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/term v0.38.0 // indirect
	golang.org/x/text v0.32.0 // indirect
)

replace github.com/mikeschinkel/go-dt => ../../go-pkgs/go-dt

replace github.com/mikeschinkel/go-dt/dtx => ../../go-pkgs/go-dt/dtx

replace github.com/mikeschinkel/go-dt/appinfo => ../../go-pkgs/go-dt/appinfo

replace github.com/mikeschinkel/go-cliutil/climenu => ../../go-pkgs/go-cliutil/climenu

replace github.com/mikeschinkel/go-cliutil => ../../go-pkgs/go-cliutil

replace github.com/mikeschinkel/go-cfgstore => ../../go-pkgs/go-cfgstore

replace github.com/mikeschinkel/go-logutil => ../../go-pkgs/go-logutil
