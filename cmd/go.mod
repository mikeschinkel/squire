module squire-cli

go 1.25.3

require github.com/mikeschinkel/squire/squirepkg v0.1.0

require (
	github.com/mikeschinkel/go-cliutil v0.2.0 // indirect
	github.com/mikeschinkel/go-dt v0.2.5 // indirect
	github.com/mikeschinkel/go-dt/appinfo v0.2.1 // indirect
	github.com/mikeschinkel/go-dt/dtx v0.2.1 // indirect
)

replace github.com/mikeschinkel/squire/squirepkg => ../squirepkg

replace github.com/mikeschinkel/go-cfgstore => ../../go-pkgs/go-cfgstore
