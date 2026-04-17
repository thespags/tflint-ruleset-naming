package rules

import "testing"

func TestFileRule(t *testing.T) {
	t.Parallel()
	runTests(t, NewFileRule())
}
