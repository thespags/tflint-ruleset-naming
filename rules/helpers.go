package rules

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

// errUnresolvableRef is returned when a format field references an expression
// that cannot be statically resolved (e.g. var.foo).
var errUnresolvableRef = errors.New("unresolvable reference")

var (
	// interpolationPattern matches {field}, {block.field}, or {?field} (optional) references in format strings.
	// Group 1 captures the optional "?" prefix, group 2 captures the field path.
	interpolationPattern = regexp.MustCompile(`\{(\??)([a-zA-Z_][a-zA-Z0-9_.]*)}`)

	// nonAlphanumeric matches any character that is not a letter or digit.
	nonAlphanumeric = regexp.MustCompile(`[^a-zA-Z0-9]+`)

	// multiUnderscore collapses consecutive underscores.
	multiUnderscore = regexp.MustCompile(`_+`)

	// namingCommentPattern matches a trailing "//naming:<value>" or "#naming:<value>" comment.
	// Captures the first non-whitespace token after "naming:", ignoring any trailing comment.
	namingCommentPattern = regexp.MustCompile(`(?://|#)\s*naming:\s*(\S+)`)
)

// sanitizeValue converts a field value into a name-safe string:
// replaces non-alphanumeric characters with _, collapses consecutive _, trims _, lowercases.
func sanitizeValue(value string) string {
	s := nonAlphanumeric.ReplaceAllString(value, "_")
	s = strings.Trim(s, "_")
	s = strings.ToLower(s)

	return s
}

// resolveFormat resolves a name format template against the attributes of a block body.
// Returns the resolved string and nil on success, or an error:
//   - nil error with empty string: attribute missing (skip silently)
//   - errUnresolvableRef: field uses an unresolvable reference like var.*
func resolveFormat(format string, body *hclsyntax.Body, src []byte) (string, error) {
	matches := interpolationPattern.FindAllStringSubmatchIndex(format, -1)
	if len(matches) == 0 {
		return format, nil
	}

	result := strings.Builder{}
	lastEnd := 0

	for _, match := range matches {
		// match[0]:match[1] is the full {...} match
		// match[2]:match[3] is group 1: optional "?" prefix
		// match[4]:match[5] is group 2: the field path
		result.WriteString(format[lastEnd:match[0]])

		optional := match[2] != match[3] // "?" was captured
		fieldPath := format[match[4]:match[5]]

		value, err := resolveFieldValue(fieldPath, body, src)
		if err != nil {
			if optional {
				lastEnd = match[1]

				continue
			}

			return "", err
		}

		if value == "" {
			if optional {
				lastEnd = match[1]

				continue
			}

			return "", nil
		}

		result.WriteString(sanitizeValue(value))

		lastEnd = match[1]
	}

	result.WriteString(format[lastEnd:])

	// Clean up separators left behind by omitted optional fields.
	resolved := multiUnderscore.ReplaceAllString(result.String(), "_")
	resolved = strings.Trim(resolved, "_")

	return resolved, nil
}

// resolveFieldValue resolves a potentially nested field path (e.g. "metadata.name")
// against a block body. Returns the string value and nil, or an error.
// A nil error with empty string means the attribute is missing (skip silently).
func resolveFieldValue(fieldPath string, body *hclsyntax.Body, src []byte) (string, error) {
	parts := strings.Split(fieldPath, ".")

	// Traverse into nested blocks for all but the last part.
	currentBody := body

	for _, blockName := range parts[:len(parts)-1] {
		found := false

		for _, block := range currentBody.Blocks {
			if block.Type == blockName {
				currentBody = block.Body
				found = true

				break
			}
		}

		if !found {
			return "", nil
		}
	}

	// Read the final attribute.
	attrName := parts[len(parts)-1]

	attr, exists := currentBody.Attributes[attrName]
	if !exists {
		return "", nil
	}

	val, diags := attr.Expr.Value(nil)
	if diags.HasErrors() || val.Type() != cty.String {
		// Check for a "// naming: <value>" comment override on the attribute line.
		if override, ok := namingComment(attr, src); ok {
			return override, nil
		}

		value, ok := resolveTraversal(attr.Expr)
		if !ok {
			return "", fmt.Errorf("%w: attribute %q is not a resolvable expression", errUnresolvableRef, fieldPath)
		}

		return value, nil
	}

	return val.AsString(), nil
}

// resolveTraversal extracts a name from a scope traversal expression by
// stripping the namespace prefix and the trailing attribute, leaving the
// meaningful identifier. Only resolves references where the name is inherently
// meaningful (module, local, resource types); skips var references since
// variable names rarely reflect runtime values.
//
//	module.foo.id               → foo
//	local.bar                   → bar
//	aws_instance.web.id         → web
//	data.aws_ami.latest.id      → aws_ami.latest
//	var.foo                     → skipped
func resolveTraversal(expr hclsyntax.Expression) (string, bool) {
	scopeExpr, ok := expr.(*hclsyntax.ScopeTraversalExpr)
	if !ok {
		return "", false
	}

	traversal := scopeExpr.Traversal
	if len(traversal) == 0 {
		return "", false
	}

	root, ok := traversal[0].(hcl.TraverseRoot)
	if !ok {
		return "", false
	}

	// Skip variable and each references — their values are dynamic and can't be statically resolved.
	if root.Name == "var" || root.Name == "each" {
		return "", false
	}

	names := make([]string, 0, len(traversal))
	for _, t := range traversal {
		switch step := t.(type) {
		case hcl.TraverseRoot:
			names = append(names, step.Name)
		case hcl.TraverseAttr:
			names = append(names, step.Name)
		}
	}

	// Strip the namespace prefix and trailing attribute.
	// module.foo.id → [module, foo, id] → [foo]
	// local.bar     → [local, bar]      → [bar]
	if len(names) >= 3 {
		names = names[1 : len(names)-1]
	} else if len(names) == 2 {
		names = names[1:]
	}

	if len(names) == 0 {
		return "", false
	}

	return strings.Join(names, "."), true
}

// namingComment extracts a "// naming: <value>" or "# naming: <value>" override
// from the source line of an attribute. Returns the value and true if found.
func namingComment(attr *hclsyntax.Attribute, src []byte) (string, bool) {
	// Find the line containing the attribute expression.
	endLine := attr.Expr.Range().End.Line
	line := sourceLine(src, endLine)

	match := namingCommentPattern.FindStringSubmatch(line)
	if match == nil {
		return "", false
	}

	return strings.TrimSpace(match[1]), true
}

// sourceLine returns the content of the given 1-based line number from src.
func sourceLine(src []byte, lineNum int) string {
	current := 1
	start := 0

	for index, b := range src {
		if current == lineNum {
			start = index

			break
		}

		if b == '\n' {
			current++
		}
	}

	end := start
	for end < len(src) && src[end] != '\n' {
		end++
	}

	return string(src[start:end])
}

// getSource extracts the string value of the "source" attribute from a block body.
func getSource(block *hclsyntax.Block) string {
	source, exists := block.Body.Attributes["source"]
	if !exists {
		return ""
	}

	val, diags := source.Expr.Value(nil)
	if diags.HasErrors() || val.Type() != cty.String {
		return ""
	}

	return val.AsString()
}
