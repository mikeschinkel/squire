module github.com/mikeschinkel/squire/squirepkg

go 1.25.3

require (
	github.com/mikeschinkel/go-cfgstore v0.0.0-00010101000000-000000000000
	github.com/mikeschinkel/go-cliutil v0.2.0
	github.com/mikeschinkel/go-dt v0.2.5
	github.com/mikeschinkel/go-dt/appinfo v0.2.1
)

require (
	github.com/mikeschinkel/go-dt/dtx v0.2.1 // indirect
	github.com/mikeschinkel/go-logutil v0.1.1 // indirect
)

replace github.com/mikeschinkel/go-cfgstore => ../../go-pkgs/go-cfgstore
