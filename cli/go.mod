module github.com/coffeemakr/threema/cli

replace github.com/coffeemakr/threema/cli/cmd => ./cmd

replace github.com/coffeemakr/threema => ../

go 1.15

require (
	github.com/coffeemakr/threema v0.0.0-00010101000000-000000000000
	github.com/coffeemakr/threema/cli/cmd v0.0.0-00010101000000-000000000000
	github.com/spf13/cobra v1.1.1
)
