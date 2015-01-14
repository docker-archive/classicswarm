package filter

import (
	"regexp"
	"strings"

	log "github.com/Sirupsen/logrus"
)

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

// Create the regex for globbing (ex: ub*t* -> ^ub.*t.*$) and match.
func match(pattern, s string) bool {
	regex := "^" + strings.Replace(pattern, "*", ".*", -1) + "$"
	matched, err := regexp.MatchString(regex, strings.ToLower(s))
	if err != nil {
		log.Error(err)
	}
	return matched
}
