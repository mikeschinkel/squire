module test

go 1.25.3

replace github.com/mikeschinkel/squire/squirepkg => ../squirepkg

replace github.com/mikeschinkel/go-fsfix => ../../go-pkgs/go-fsfix

replace github.com/mikeschinkel/go-cliutil => ../../go-pkgs/go-cliutil

require (
	github.com/mikeschinkel/go-cfgstore v0.4.1
	github.com/mikeschinkel/go-cliutil v0.3.0
	github.com/mikeschinkel/go-dt v0.4.1
	github.com/mikeschinkel/go-dt/appinfo v0.2.1
	github.com/mikeschinkel/go-fsfix v0.2.2
	github.com/mikeschinkel/go-testutil v0.3.0
	github.com/mikeschinkel/squire/squirepkg v0.0.0-00010101000000-000000000000
	github.com/stretchr/testify v1.11.1
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/jedib0t/go-pretty/v6 v6.7.7 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/mikeschinkel/go-dt/dtx v0.3.0 // indirect
	github.com/mikeschinkel/go-logutil v0.2.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	golang.org/x/mod v0.31.0 // indirect
	golang.org/x/sys v0.30.0 // indirect
	golang.org/x/text v0.22.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/mikeschinkel/go-dt => ../../go-pkgs/go-dt

replace github.com/mikeschinkel/go-testutil => ../../go-pkgs/go-testutil

replace github.com/mikeschinkel/go-dt/dtx => ../../go-pkgs/go-dt/dtx

replace github.com/mikeschinkel/go-cfgstore => ../../go-pkgs/go-cfgstore
