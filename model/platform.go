package model

import (
	"fmt"
	"strings"
)

// TODO: Support more than X?
type Platform string

const (
	PlatformX        Platform = "X"
	PlatformReddit   Platform = "REDDIT"
	PlatformMastadon Platform = "MASTADON" // typo'd consistently with everything else
	// TODO: Add more as needed
)

func ParsePlatform(s string) (Platform, error) {
	switch strings.ToUpper(s) {
	case string(PlatformX):
		return PlatformX, nil
	case string(PlatformReddit):
		return PlatformReddit, nil
	case string(PlatformMastadon):
		return PlatformMastadon, nil
	default:
		return PlatformX, fmt.Errorf("unknown platform: %s", s)
	}
}
