package auth

import (
	"fmt"
	"strings"
)

func validatePattern(pattern string) error {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return fmt.Errorf("pattern is empty")
	}
	if strings.Count(pattern, "*") > 1 {
		return fmt.Errorf("only one wildcard is supported")
	}
	return nil
}

func matches(pattern, path string) bool {
	pattern = strings.TrimSpace(pattern)
	star := strings.IndexByte(pattern, '*')
	if star < 0 {
		return path == pattern
	}
	prefix, suffix := pattern[:star], pattern[star+1:]
	return len(path) >= len(prefix)+len(suffix) && strings.HasPrefix(path, prefix) && strings.HasSuffix(path, suffix)
}
