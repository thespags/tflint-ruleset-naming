package rules

import "testing"

func TestNameRule(t *testing.T) {
	t.Parallel()
	runTests(t, NewNameRule())
}
