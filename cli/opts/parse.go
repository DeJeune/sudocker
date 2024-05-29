package opts

import (
	"os"
	"strings"
)

func ReadKVStrings(files []string, override []string) ([]string, error) {
	return readKVStrings(files, override, nil)
}

func ReadKVEnvStrings(files []string, ovveride []string) ([]string, error) {
	return readKVStrings(files, ovveride, os.LookupEnv)
}

func readKVStrings(files []string, override []string, emptyFn func(string) (string, bool)) ([]string, error) {
	var variables []string
	for _, ef := range files {
		parsedVars, err := parseKeyValueFile(ef, emptyFn)
		if err != nil {
			return nil, err
		}
		variables = append(variables, parsedVars...)
	}
	// parse the '-e' and '--env' after, to allow override
	variables = append(variables, override...)

	return variables, nil
}

// convert "key=value" to {"key":"value"}
func ConvertKVStringsToMap(values []string) map[string]string {
	result := make(map[string]string, len(values))
	for _, value := range values {
		k, v, _ := strings.Cut(value, "=")
		result[k] = v
	}
	return result
}

func ConvertKVStringsToMapWithNil(values []string) map[string]*string {
	result := make(map[string]*string, len(values))
	for _, value := range values {
		k, v, ok := strings.Cut(value, "=")
		if !ok {
			result[k] = nil
		} else {
			result[k] = &v
		}
	}
	return result
}
