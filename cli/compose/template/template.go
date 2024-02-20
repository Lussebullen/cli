// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.19

package template

import (
	"fmt"
	"os"
	"sort"
	"regexp"
	"strings"
)

const (
	delimiter = "\\$"
	subst     = "[_a-z][_a-z0-9]*(?::?[-?][^}]*)?"
)

var defaultPattern = regexp.MustCompile(fmt.Sprintf(
	"%s(?i:(?P<escaped>%s)|(?P<named>%s)|{(?P<braced>%s)}|(?P<invalid>))",
	delimiter, delimiter, subst, subst,
))

// DefaultSubstituteFuncs contains the default SubstituteFunc used by the docker cli
var DefaultSubstituteFuncs = []SubstituteFunc{
	softDefault,
	hardDefault,
	requiredNonEmpty,
	required,
}

// InvalidTemplateError is returned when a variable template is not in a valid
// format
type InvalidTemplateError struct {
	Template string
}

func (e InvalidTemplateError) Error() string {
	return fmt.Sprintf("Invalid template: %#v", e.Template)
}

// Mapping is a user-supplied function which maps from variable names to values.
// Returns the value as a string and a bool indicating whether
// the value is present, to distinguish between an empty string
// and the absence of a value.
type Mapping func(string) (string, bool)

// SubstituteFunc is a user-supplied function that apply substitution.
// Returns the value as a string, a bool indicating if the function could apply
// the substitution and an error.
type SubstituteFunc func(string, Mapping) (string, bool, error)

// SubstituteWith subsitute variables in the string with their values.
// It accepts additional substitute function.
func SubstituteWith(template string, mapping Mapping, pattern *regexp.Regexp, subsFuncs ...SubstituteFunc) (string, error) {
	var err error
	result := pattern.ReplaceAllStringFunc(template, func(substring string) string {
		matches := pattern.FindStringSubmatch(substring)
		groups := matchGroups(matches, pattern)
		if escaped := groups["escaped"]; escaped != "" {
			return escaped
		}

		substitution := groups["named"]
		if substitution == "" {
			substitution = groups["braced"]
		}

		if substitution == "" {
			err = &InvalidTemplateError{Template: template}
			return ""
		}

		for _, f := range subsFuncs {
			var (
				value   string
				applied bool
			)
			value, applied, err = f(substitution, mapping)
			if err != nil {
				return ""
			}
			if !applied {
				continue
			}
			return value
		}

		value, _ := mapping(substitution)
		return value
	})

	return result, err
}

// Substitute variables in the string with their values
func Substitute(template string, mapping Mapping) (string, error) {
	return SubstituteWith(template, mapping, defaultPattern, DefaultSubstituteFuncs...)
}

// ExtractVariables returns a map of all the variables defined in the specified
// composefile (dict representation) and their default value if any.
func ExtractVariables(configDict map[string]any, pattern *regexp.Regexp) map[string]string {
	if pattern == nil {
		pattern = defaultPattern
	}
	return recurseExtract(configDict, pattern)
}

func recurseExtract(value any, pattern *regexp.Regexp) map[string]string {
	m := map[string]string{}

	switch value := value.(type) {
	case string:
		if values, is := extractVariable(value, pattern); is {
			for _, v := range values {
				m[v.name] = v.value
			}
		}
	case map[string]any:
		for _, elem := range value {
			submap := recurseExtract(elem, pattern)
			for key, value := range submap {
				m[key] = value
			}
		}

	case []any:
		for _, elem := range value {
			if values, is := extractVariable(elem, pattern); is {
				for _, v := range values {
					m[v.name] = v.value
				}
			}
		}
	}

	return m
}

type extractedValue struct {
	name  string
	value string
}

var extractVariablesCoverage = map[int]bool{
	1: false,
	2: false,
	3: false,
	4: false,
	5: false,
	6: false,
	7: false,
	8: false,
	9: false,
	10: false,
	11: false,
	12: false,
}
func extractVariable(value any, pattern *regexp.Regexp) ([]extractedValue, bool) {
	sValue, ok := value.(string)
	if !ok {
		extractVariablesCoverage[1] = true
		//extractVariablesCoverage["ofif"] = true 
		return []extractedValue{}, false
	} else  {
		extractVariablesCoverage[2] = true
	}
	matches := pattern.FindAllStringSubmatch(sValue, -1)
	if len(matches) == 0 {
		extractVariablesCoverage[3] = true
		return []extractedValue{}, false
	} else {
		extractVariablesCoverage[4] = true
	}
	values := []extractedValue{}
	for _, match := range matches {
		groups := matchGroups(match, pattern)
		if escaped := groups["escaped"]; escaped != "" {
			extractVariablesCoverage[5] = true
			continue
		} else {
			extractVariablesCoverage[6] = true}
		val := groups["named"]
		if val == "" {
			extractVariablesCoverage[7] = true
			val = groups["braced"]
		} else {
			extractVariablesCoverage[8] = true
		}
		name := val
		var defaultValue string
		switch {
		case strings.Contains(val, ":?"):
			extractVariablesCoverage[9] = true
			name, _ = partition(val, ":?")
		case strings.Contains(val, "?"):
			extractVariablesCoverage[10] = true
			name, _ = partition(val, "?")
		case strings.Contains(val, ":-"):
			extractVariablesCoverage[11] = true
			name, defaultValue = partition(val, ":-")
		case strings.Contains(val, "-"):
			extractVariablesCoverage[12] = true
			name, defaultValue = partition(val, "-")
		}
		values = append(values, extractedValue{name: name, value: defaultValue})
	}

	writeCoverageToFile("extractVariablesCoverage.txt", extractVariablesCoverage)

	return values, len(values) > 0
}

func writeCoverageToFile(filename string, data map[int]bool) {
	f, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	var builder strings.Builder
	var sum float64
	type keyValue struct {
		key int
		val bool
	}

	var kv []keyValue

	for k, v := range data {
		if v {
			sum++
		}
		kv = append(kv, keyValue{k, v})
	}

	sort.Slice(kv, func(i, j int) bool {
		return kv[i].key < kv[j].key
	})

	for _, v := range kv {
		builder.WriteString(fmt.Sprintf("Branch %d\t -\t %t\n", v.key, v.val))
	}

	builder.WriteString(fmt.Sprintf("\nCoverage Percentage - %2.f%%", (sum / float64(len(data)))*100))

	f.WriteString(builder.String())
}

// Soft default (fall back if unset or empty)
func softDefault(substitution string, mapping Mapping) (string, bool, error) {
	sep := ":-"
	if !strings.Contains(substitution, sep) {
		return "", false, nil
	}
	name, defaultValue := partition(substitution, sep)
	value, ok := mapping(name)
	if !ok || value == "" {
		return defaultValue, true, nil
	}
	return value, true, nil
}

// Hard default (fall back if-and-only-if empty)
func hardDefault(substitution string, mapping Mapping) (string, bool, error) {
	sep := "-"
	if !strings.Contains(substitution, sep) {
		return "", false, nil
	}
	name, defaultValue := partition(substitution, sep)
	value, ok := mapping(name)
	if !ok {
		return defaultValue, true, nil
	}
	return value, true, nil
}

func requiredNonEmpty(substitution string, mapping Mapping) (string, bool, error) {
	return withRequired(substitution, mapping, ":?", func(v string) bool { return v != "" })
}

func required(substitution string, mapping Mapping) (string, bool, error) {
	return withRequired(substitution, mapping, "?", func(_ string) bool { return true })
}

func withRequired(substitution string, mapping Mapping, sep string, valid func(string) bool) (string, bool, error) {
	if !strings.Contains(substitution, sep) {
		return "", false, nil
	}
	name, errorMessage := partition(substitution, sep)
	value, ok := mapping(name)
	if !ok || !valid(value) {
		return "", true, &InvalidTemplateError{
			Template: fmt.Sprintf("required variable %s is missing a value: %s", name, errorMessage),
		}
	}
	return value, true, nil
}

func matchGroups(matches []string, pattern *regexp.Regexp) map[string]string {
	groups := make(map[string]string)
	for i, name := range pattern.SubexpNames()[1:] {
		groups[name] = matches[i+1]
	}
	return groups
}

// Split the string at the first occurrence of sep, and return the part before the separator,
// and the part after the separator.
//
// If the separator is not found, return the string itself, followed by an empty string.
func partition(s, sep string) (string, string) {
	k, v, ok := strings.Cut(s, sep)
	if !ok {
		return s, ""
	}
	return k, v
}
