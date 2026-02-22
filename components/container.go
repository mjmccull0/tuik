package components

import "strings"

func RenderContainer(renderedChildren []string) string {
	return strings.Join(renderedChildren, "\n")
}
