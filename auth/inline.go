package auth

import (
	"fmt"

	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
)

const (
	AUTH_DICT = "auth-dict"
)

func init() {
	authProviders["inline"] = InlineAuthenticator
}

func InlineAuthenticator(env map[string]string) (err error) {
	adStr, found := env[AUTH_DICT]

	if !found {
		return fmt.Errorf("[inline-auth] No auth JSON dictionary available. Please set one using --auth-dict.\n")
	}

	// Remove the auth dictionary from the env passed along
	delete(env, AUTH_DICT)

	var ad map[string]string
	if err := json.Unmarshal([]byte(adStr), &ad); err != nil {
		return fmt.Errorf("[inline-auth] Decoding auth JSON dictionary failed. Reason: %v\n", err)
	} else if username, found := env[AUTH_USERNAME]; !found {
		return fmt.Errorf("[inline-auth] Auth failed. No user specified.\n")
	} else if password, found := env[AUTH_PASSWORD]; !found {
		return fmt.Errorf("[inline-auth] Auth failed. No password specified.\n")
	} else if storedHash, found := ad[username]; found {
		// Remove the password from the env passed along
		delete(env, AUTH_PASSWORD)

		hashBytes := sha1.Sum([]byte(password))
		if hashStr := hex.EncodeToString(hashBytes[:]); storedHash != hashStr {
			return fmt.Errorf("[inline-auth] Auth failed. Bad credentials.\n")
		}
	}

	return
}
