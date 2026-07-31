package main

import (
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// versionPattern matches the version identifiers Paper publishes:
//
//	legacy releases      1.8.8, 1.20, 1.21.4
//	new scheme           26.2, 26.2.1
//	pre-releases and rcs 1.13-pre7, 1.21.11-rc3, 26.2-rc-2
//
// Mojang switched to a year.release[.patch] scheme in 2026, so the leading
// number is not pinned to 1.
var versionPattern = regexp.MustCompile(`^\d+\.\d+(\.\d+)?(-[0-9A-Za-z][0-9A-Za-z.-]*)?$`)

func isValidVersion(version string) bool {
	return versionPattern.MatchString(version)
}

// compareVersions orders two version identifiers, returning 1, 0 or -1 for
// v1 greater than, equal to, or less than v2. Numeric components compare
// numerically, and a pre-release sorts below the release it leads up to
// (1.21.11-rc3 < 1.21.11).
func compareVersions(v1, v2 string) int {
	base1, pre1 := splitPreRelease(v1)
	base2, pre2 := splitPreRelease(v2)

	if c := compareNumericParts(base1, base2); c != 0 {
		return c
	}

	switch {
	case pre1 == "" && pre2 == "":
		return 0
	case pre1 == "":
		return 1
	case pre2 == "":
		return -1
	}
	return comparePreRelease(pre1, pre2)
}

// splitPreRelease separates "1.21.11-rc3" into "1.21.11" and "rc3".
func splitPreRelease(version string) (base, pre string) {
	if i := strings.IndexByte(version, '-'); i >= 0 {
		return version[:i], version[i+1:]
	}
	return version, ""
}

// compareNumericParts compares dot-separated numbers, treating missing
// trailing components as zero so 1.20 and 1.20.0 are equal.
func compareNumericParts(v1, v2 string) int {
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		var p1, p2 int
		if i < len(parts1) {
			p1, _ = strconv.Atoi(parts1[i])
		}
		if i < len(parts2) {
			p2, _ = strconv.Atoi(parts2[i])
		}

		if p1 > p2 {
			return 1
		} else if p1 < p2 {
			return -1
		}
	}
	return 0
}

// comparePreRelease orders two pre-release suffixes token by token, so that
// "pre4" < "pre10" < "rc1" and "rc-2" < "rc-10".
func comparePreRelease(pre1, pre2 string) int {
	t1 := preReleaseTokens(pre1)
	t2 := preReleaseTokens(pre2)

	for i := 0; i < len(t1) && i < len(t2); i++ {
		a, b := t1[i], t2[i]
		numA, errA := strconv.Atoi(a)
		numB, errB := strconv.Atoi(b)

		switch {
		case errA == nil && errB == nil:
			if numA != numB {
				return sign(numA - numB)
			}
		case a != b:
			return strings.Compare(a, b)
		}
	}

	// A shorter suffix is the lower one: "rc" < "rc2".
	return sign(len(t1) - len(t2))
}

// preReleaseTokens splits a suffix into alternating alphabetic and numeric
// runs, dropping separators, so "rc3", "rc-3" and "rc.3" all tokenise the
// same way and numbers can be compared as numbers.
func preReleaseTokens(s string) []string {
	var tokens []string
	start := 0
	for i := 1; i <= len(s); i++ {
		if i < len(s) && charClass(s[i]) == charClass(s[start]) {
			continue
		}
		if charClass(s[start]) != classSeparator {
			tokens = append(tokens, strings.ToLower(s[start:i]))
		}
		start = i
	}
	return tokens
}

const (
	classSeparator = iota
	classDigit
	classAlpha
)

func charClass(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return classDigit
	case (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z'):
		return classAlpha
	default:
		return classSeparator
	}
}

func sign(n int) int {
	switch {
	case n > 0:
		return 1
	case n < 0:
		return -1
	default:
		return 0
	}
}

// getVersionsFromJars lists the Paper versions already downloaded into the
// working directory, newest first.
func getVersionsFromJars() []string {
	files, err := filepath.Glob("paper-*.jar")
	if err != nil {
		return nil
	}

	versions := make([]string, 0, len(files))
	for _, file := range files {
		version := strings.TrimSuffix(strings.TrimPrefix(file, "paper-"), ".jar")
		if isValidVersion(version) {
			versions = append(versions, version)
		}
	}

	sortVersionsDescending(versions)
	return versions
}

func sortVersionsDescending(versions []string) {
	sort.Slice(versions, func(i, j int) bool {
		return compareVersions(versions[i], versions[j]) > 0
	})
}
