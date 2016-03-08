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

// Expr is exported
type Expr struct {
	Key      string
	Operator int
	Value    string
	IsSoft   bool
}

// ParseExprs is exported to parse the filters.
func ParseExprs(env []string) ([]Expr, error) {
	exprs := []Expr{}
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
					exprs = append(exprs, Expr{Key: parts[0], Operator: i, Value: strings.TrimLeft(parts[1], "~"), IsSoft: isSoft(parts[1])})
				} else {
					exprs = append(exprs, Expr{Key: parts[0], Operator: i})
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

// Match is exported.
func (e *Expr) Match(whats ...string) bool {
	var (
		pattern string
		match   bool
		err     error
	)

	if e.Value[0] == '/' && e.Value[len(e.Value)-1] == '/' {
		// regexp
		pattern = e.Value[1 : len(e.Value)-1]
	} else {
		// simple match, create the regex for globbing (ex: ub*t* -> ^ub.*t.*$) and match.
		pattern = "^" + strings.Replace(e.Value, "*", ".*", -1) + "$"
	}

	for _, what := range whats {
		if match, err = regexp.MatchString(pattern, what); match {
			break
		} else if err != nil {
			log.Error(err)
		}
	}

	switch e.Operator {
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
