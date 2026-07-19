package scripts

import (
	"regexp"
	"strings"
)

// makeTarget matches a top-level Makefile rule: "migrate:" or "build: deps".
// Pattern rules (%), variables (=) and indented lines are excluded.
var makeTarget = regexp.MustCompile(`^([A-Za-z0-9][A-Za-z0-9._-]*)\s*:(?:[^=]|$)`)

// makeIgnore are bookkeeping targets that aren't useful as buttons.
var makeIgnore = map[string]bool{
	".PHONY": true, "default": true, "all": true, "help": true,
}

func detectMakefile(dir string) []Found {
	name := "Makefile"
	data, ok := readFile(dir, name)
	if !ok {
		name = "makefile"
		if data, ok = readFile(dir, name); !ok {
			return nil
		}
	}

	var out []Found
	for _, line := range strings.Split(string(data), "\n") {
		if line == "" || line[0] == '\t' || line[0] == '#' || line[0] == ' ' {
			continue
		}
		m := makeTarget.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		target := m[1]
		if makeIgnore[target] || isLongRunning(target) {
			continue
		}
		out = append(out, Found{
			Name:    label(target),
			Command: "make " + target,
			Source:  name,
		})
	}
	return out
}
