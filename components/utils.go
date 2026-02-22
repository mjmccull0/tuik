package components 

import "strings"

func Interpolate(template string, values map[string]string) string {
	for key, val := range values {
		template = strings.ReplaceAll(template, "{{."+key+"}}", val)
	}
	return template
}

func GetFirstFocusable(children []Component) int {
	for i, child := range children {
		if child.IsFocusable() {
			return i
		}
	}
	return 0
}
