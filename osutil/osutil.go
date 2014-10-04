package osutil

import (
	"errors"
	"os"
	"strings"
)

func ResolveFilePathInEnv(env, path string) (string, error) {
	envEntries := strings.Split(os.Getenv(env), ":")
	var resolvedPath string
	for _, gopath := range envEntries {
		resolvedPath = gopath + path
		if _, err := os.Stat(resolvedPath); err == nil {
			return resolvedPath, nil
		}
	}
	return "", errors.New("unable to resolve file path")
}
