module github.com/mikeschinkel/squire/squirepkg

go 1.25.3

require (
	github.com/mikeschinkel/go-cfgstore v0.4.0
	github.com/mikeschinkel/go-cliutil v0.3.0
	github.com/mikeschinkel/go-dt v0.3.3
	github.com/mikeschinkel/go-dt/appinfo v0.2.1
	github.com/mikeschinkel/go-logutil v0.2.1
)

require github.com/mikeschinkel/go-dt/dtx v0.2.1 // indirect

replace github.com/mikeschinkel/go-dt => ../../go-pkgs/go-dt
