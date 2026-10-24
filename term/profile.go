package term

import (
	"os"
	"strings"
)

// ColorProfile describes how many colors the terminal can display.
type ColorProfile uint8

const (
	ProfileNone ColorProfile = iota
	ProfileANSI
	Profile256
	ProfileTrueColor
)

// DetectProfile inspects an environment map (e.g. from os.Environ) and
// reports the terminal's color capability. NO_COLOR wins over everything;
// COLORTERM and TERM provide the remaining signal.
func DetectProfile(env map[string]string) ColorProfile {
	if v, ok := env["NO_COLOR"]; ok && v != "" {
		return ProfileNone
	}
	switch strings.ToLower(env["COLORTERM"]) {
	case "truecolor", "24bit":
		return ProfileTrueColor
	}
	switch term := env["TERM"]; {
	case strings.Contains(term, "truecolor"), strings.Contains(term, "direct"):
		return ProfileTrueColor
	case strings.Contains(term, "256color"):
		return Profile256
	case term == "" || term == "dumb":
		return ProfileNone
	default:
		return ProfileANSI
	}
}

// ProfileFromEnv detects the color profile of the actual process environment.
func ProfileFromEnv() ColorProfile {
	env := map[string]string{}
	for _, kv := range os.Environ() {
		if i := strings.IndexByte(kv, '='); i > 0 {
			env[kv[:i]] = kv[i+1:]
		}
	}
	return DetectProfile(env)
}
