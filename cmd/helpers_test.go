package cmd_test

// check fails the surrounding test setup on error.
func check(e error) {
	if e != nil {
		panic(e)
	}
}
