package opts

import (
	"errors"
	"os"
	"strings"
)

func ValidateEnv(val string) (string, error) {
	k, _, hasValue := strings.Cut(val, "=") //key=value
	if k == "" {
		return "", errors.New("invalid environment variable: " + val)
	}
	if hasValue {
		// val contains a "=" (but value may be an empty string)
		return val, nil
	}
	if envVal, ok := os.LookupEnv(k); ok {
		return k + "=" + envVal, nil
	}
	return val, nil
}
