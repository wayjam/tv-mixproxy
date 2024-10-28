package mixer

import "regexp"

func compileRegex(pattern string) *regexp.Regexp {
	if pattern == "" {
		return nil
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		// 如果正则表达式无效，返回nil
		return nil
	}
	return re
}

func matchFilter(value string, includeRegex, excludeRegex *regexp.Regexp) bool {
	if includeRegex != nil && !includeRegex.MatchString(value) {
		return false
	}
	if excludeRegex != nil && excludeRegex.MatchString(value) {
		return false
	}
	return true
}
