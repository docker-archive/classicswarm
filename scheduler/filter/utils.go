package filter

import (
	"fmt"
	"regexp"
	"strings"

	log "github.com/Sirupsen/logrus"
)

type opWithValue []string

func extractEnv(key string, env []string) (map[string]opWithValue, error) {
	ops := []string{"==", "!=", ">=", "<="}
	values := make(map[string]opWithValue)
	for _, e := range env {
		if strings.HasPrefix(e, key+":") {
			entry := strings.TrimPrefix(e, key+":")
			found := false
			for _, op := range ops {
				if strings.Contains(entry, op) {
					// split with the op
					parts := strings.SplitN(entry, op, 2)

					// validate key
					// allow alpha-numeric
					matched, err := regexp.MatchString(`^(?i)[a-z_][a-z0-9\-_]+$`, parts[0])
					if err != nil {
						return nil, err
					}
					if matched == false {
						return nil, fmt.Errorf("Key '%s' is invalid", parts[0])
					}

					if len(parts) == 2 {

						// validate value
						// allow leading = in case of using ==
						// allow * for globbing
						// allow regexp
						matched, err := regexp.MatchString(`^(?i)[=!\/]?[a-z0-9:\-_\.\*/\(\)\?\+\[\]\\\^\$]+$`, parts[1])
						if err != nil {
							return nil, err
						}
						if matched == false {
							return nil, fmt.Errorf("Value '%s' is invalid", parts[1])
						}
						values[strings.ToLower(parts[0])] = opWithValue{op, parts[1]}
					} else {
						values[strings.ToLower(parts[0])] = opWithValue{op, ""}
					}

					found = true
					break // found an op, move to next entry
				}
			}
			if !found {
				return nil, fmt.Errorf("One of operator ==, !=, >=, <= is expected")
			}
		}
	}
	return values, nil
}

// Create the regex for globbing (ex: ub*t* -> ^ub.*t.*$) and match.
// If useRegex is true, the pattern will be used directly
func internalMatch(pattern, s string) bool {
	regex := pattern
	useRegex := false
	if strings.HasPrefix(pattern, "/") && strings.HasSuffix(pattern, "/") {
		log.Debugf("regex detected")
		regex = strings.TrimPrefix(strings.TrimSuffix(pattern, "/"), "/")
		useRegex = true
	}

	if !useRegex {
		regex = "^" + strings.Replace(pattern, "*", ".*", -1) + "$"
	}
	matched, err := regexp.MatchString(regex, s)
	if err != nil {
		log.Error(err)
	}
	return matched
}

func match(val opWithValue, what string) bool {
	op, v := val[0], val[1]
	if op == ">=" && what >= v {
		return true
	} else if op == "<=" && what <= v {
		return true
	} else {
		matchResult := internalMatch(v, what)
		if (op == "!=" && !matchResult) || (op == "==" && matchResult) {
			return true
		}
	}

	return false
}
