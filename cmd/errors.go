package cmd

import "errors"

// errQuiet ends a command with a failing exit code when the reason has already
// been shown to the user, so it is not printed a second time.
var errQuiet = errors.New("")
