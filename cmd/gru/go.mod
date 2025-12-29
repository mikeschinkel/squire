module github.com/mikeschinkel/squire/gru

go 1.25.5

require (
	github.com/mikeschinkel/go-cliutil v0.0.0
	github.com/mikeschinkel/go-dt v0.0.0
	github.com/mikeschinkel/squire/gru/grumod v0.0.0
	github.com/mikeschinkel/squire/squirepkg v0.0.0
)

replace github.com/mikeschinkel/squire/gru/grumod => ./grumod
replace github.com/mikeschinkel/squire/squirepkg => ./../../squirepkg

replace github.com/mikeschinkel/go-cliutil => ./../../../go-pkgs/go-cliutil

replace github.com/mikeschinkel/go-dt => ./../../../go-pkgs/go-dt

replace github.com/mikeschinkel/go-dt/dtx => ./../../../go-pkgs/go-dt/dtx

replace github.com/mikeschinkel/go-cfgstore => ./../../../go-pkgs/go-cfgstore
