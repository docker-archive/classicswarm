package filter

import (
	"regexp"
	"strings"

	log "github.com/Sirupsen/logrus"
)

<<<<<<< HEAD
=======
type comparison int

const (
	equ = comparison(iota)
	neg
	gte
	lte
)

func parse(k, v string) (string, string, comparison, bool) {
	// default comparison mode
	mode := equ

	// support case of constraint:k==v
	// with "=", it's possible for an entry "k==v" to be split as:
	// 1. "k=" as key and "v" as value
	// 2. "k" as key and "=v" as value
	// Just to make sure it cover these cases.
	if strings.HasSuffix(k, "=") {
		k = strings.TrimSuffix(k, "=")
	} else if strings.HasPrefix(v, "=") {
		v = strings.TrimPrefix(v, "=")
	}

	if strings.HasPrefix(v, "!") {
		log.Debugf("negate detected in value")
		v = strings.TrimPrefix(v, "!")
		mode = neg
	} else if strings.HasSuffix(k, "!") {
		log.Debugf("negate detected in key")
		k = strings.TrimSuffix(k, "!")
		mode = neg
	} else {
		if strings.HasSuffix(k, ">") {
			log.Debugf("gt (>) detected in key")
			k = strings.TrimSuffix(k, ">")
			mode = gte
		} else if strings.HasSuffix(k, "<") {
			log.Debugf("lt (<) detected in key")
			k = strings.TrimSuffix(k, "<")
			mode = lte
		}
	}

	useRegex := false
	if strings.HasPrefix(v, "/") && strings.HasSuffix(v, "/") {
		log.Debugf("regex detected")
		v = strings.TrimPrefix(strings.TrimSuffix(v, "/"), "/")
		useRegex = true
	}

	return k, v, mode, useRegex
}

>>>>>>> add double equals comparison
func extractEnv(key string, env []string) map[string]string {
	values := make(map[string]string)
	for _, e := range env {
		if strings.HasPrefix(e, key+":") {
			value := strings.TrimPrefix(e, key+":")
			parts := strings.SplitN(value, "=", 2)
			if len(parts) == 2 {
				values[strings.ToLower(parts[0])] = strings.ToLower(parts[1])
			} else {
				values[strings.ToLower(parts[0])] = ""
			}
		}
	}
	return values
}

<<<<<<< HEAD
// Create the regex for globbing (ex: ub*t* -> ^ub.*t.*$) and match.
func match(pattern, s string) bool {
	regex := "^" + strings.Replace(pattern, "*", ".*", -1) + "$"
	matched, err := regexp.MatchString(regex, strings.ToLower(s))
=======
// Create the regex for globbing (ex: ub*t* -> ^ub.*t.*$)
// and match.
func match(pattern, s string, useRegex bool) bool {
	regex := pattern
	if !useRegex {
		regex = "^" + strings.Replace(pattern, "*", ".*", -1) + "$"
	}
	matched, err := regexp.MatchString(regex, s)
>>>>>>> add double equals comparison
	if err != nil {
		log.Error(err)
	}
	return matched
}
