package opts

import "os"

func ParseEnvFile(filename string) ([]string, error) {
	return parseKeyValueFile(filename, os.LookupEnv)
}
