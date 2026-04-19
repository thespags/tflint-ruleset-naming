package project

import "fmt"

// Name is the name of the plugin.
const Name string = "naming"

// Version is the ruleset version.
const Version string = "0.0.2"

// ReferenceLink returns the rule reference link.
func ReferenceLink(name string) string {
	return fmt.Sprintf(
		"https://github.com/thespags/tflint-ruleset-naming/blob/v%s/docs/%s.md",
		Version,
		name,
	)
}

// RuleName returns the name of the rule.
func RuleName(id string) string {
	return fmt.Sprintf(
		"%s_%s",
		Name,
		id,
	)
}
