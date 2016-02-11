package filter

import (
	"fmt"
	"regexp"
	"strings"

	log "github.com/Sirupsen/logrus"
)

const (
	// EQ is exported
	EQ = iota
	// NOTEQ is exported
	NOTEQ
)

// OPERATORS is exported
var OPERATORS = []string{"==", "!="}

type expr struct {
	key      string
	operator int
	value    string
	isSoft   bool
}

func parseExprs(env []string) ([]expr, error) {
	exprs := []expr{}
	for _, e := range env {
		found := false
		for i, op := range OPERATORS {
			if strings.Contains(e, op) {
				// split with the op
				parts := strings.SplitN(e, op, 2)

				// validate key
				// allow alpha-numeric
				matched, err := regexp.MatchString(`^(?i)[a-z_][a-z0-9\-_.]+$`, parts[0])
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
					matched, err := regexp.MatchString(`^(?i)[=!\/]?(~)?[a-z0-9:\-_\s\.\*/\(\)\?\+\[\]\\\^\$\|]+$`, parts[1])
					if err != nil {
						return nil, err
					}
					if matched == false {
						return nil, fmt.Errorf("Value '%s' is invalid", parts[1])
					}
					exprs = append(exprs, expr{key: parts[0], operator: i, value: strings.TrimLeft(parts[1], "~"), isSoft: isSoft(parts[1])})
				} else {
					exprs = append(exprs, expr{key: parts[0], operator: i})
				}

				found = true
				break // found an op, move to next entry
			}
		}
		if !found {
			return nil, fmt.Errorf("One of operator ==, != is expected")
		}
	}
	return exprs, nil
}

func (e *expr) Match(whats ...string) bool {
	var (
		pattern string
		match   bool
		err     error
	)

	if e.value[0] == '/' && e.value[len(e.value)-1] == '/' {
		// regexp
		pattern = e.value[1 : len(e.value)-1]
	} else {
		// simple match, create the regex for globbing (ex: ub*t* -> ^ub.*t.*$) and match.
		pattern = "^" + strings.Replace(e.value, "*", ".*", -1) + "$"
	}

	for _, what := range whats {
		if match, err = regexp.MatchString(pattern, what); match {
			break
		} else if err != nil {
			log.Error(err)
		}
	}

	switch e.operator {
	case EQ:
		return match
	case NOTEQ:
		return !match
	}

	return false
}

func isSoft(value string) bool {
	if value[0] == '~' {
		return true
	}
	return false
}
