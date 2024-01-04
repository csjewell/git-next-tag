/*
Copyright © 2023 Curtis Jewell <golang@curtisjewell.name>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

// Package semver implements the version parsing.
package semver

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var Regexp = func() *regexp.Regexp {
	rx, _ := regexp.Compile(`v?(\d+)[.](\d+)[.](\d+)(?:-(alpha|beta|gamma|rc)[.](\d+))?(?:[+](pre))?`)
	return rx
}()

// VersionSegment specifies what segment of the version is being changed.
type VersionSegment int

const (
	None VersionSegment = iota
	Major
	Minor
	Patch
	Alpha
	Beta
	Gamma
	RC
	Pre
)

var vsName = []string{"", "major", "minor", "patch", "alpha", "beta", "gamma", "rc", "pre"}

// String is provided in order to satisfy the fmt.Stringer interface.
func (vs VersionSegment) String() string {
	return vsName[vs]
}

// ParsedVersion is a data type that represents a semver version within the limits this module uses.
type ParsedVersion struct {
	major         int
	minor         int
	patch         int
	lower         int
	lowerCategory VersionSegment
	isPre         bool
}

// ParseVersion attempts to parse a version string.
func ParseVersion(v string) *ParsedVersion {
	matches := Regexp.FindStringSubmatch(v)
	if len(matches) == 0 {
		return nil
	}
	var pv ParsedVersion
	if strings.ToLower(matches[6]) == "pre" {
		pv.isPre = true
	}
	if strings.ToLower(matches[4]) == "alpha" {
		pv.lower, _ = strconv.Atoi(matches[5])
		pv.lowerCategory = Alpha
	}
	if strings.ToLower(matches[4]) == "beta" {
		pv.lower, _ = strconv.Atoi(matches[5])
		pv.lowerCategory = Beta
	}
	if strings.ToLower(matches[4]) == "gamma" {
		pv.lower, _ = strconv.Atoi(matches[5])
		pv.lowerCategory = Gamma
	}
	if strings.ToLower(matches[4]) == "rc" {
		pv.lower, _ = strconv.Atoi(matches[5])
		pv.lowerCategory = RC
	}
	pv.major, _ = strconv.Atoi(matches[1])
	pv.minor, _ = strconv.Atoi(matches[2])
	pv.patch, _ = strconv.Atoi(matches[3])

	return &pv
}

// String is provided in order to satisfy the fmt.Stringer interface.
func (pv ParsedVersion) String() string {
	s := fmt.Sprint(pv.major, ".", pv.minor, ".", pv.patch)
	if pv.lowerCategory != None {
		s += fmt.Sprint("-", pv.lowerCategory, ".", pv.lower)
	}
	if pv.isPre {
		s += "-pre"
	}
	return s
}

func (pv ParsedVersion) IncrementVersion(vs VersionSegment) (*ParsedVersion, error) {
	var pvNext ParsedVersion
	switch vs {
	case Major:
		pvNext = ParsedVersion{
			major:         pv.major + 1,
			minor:         0,
			patch:         0,
			lower:         0,
			lowerCategory: None,
			isPre:         false,
		}
		return &pvNext, nil
	case Minor:
		pvNext = ParsedVersion{
			major:         pv.major,
			minor:         pv.minor + 1,
			patch:         0,
			lower:         0,
			lowerCategory: None,
			isPre:         false,
		}
		return &pv, nil
	case Patch:
		pvNext = ParsedVersion{
			major:         pv.major,
			minor:         pv.minor,
			patch:         pv.patch + 1,
			lower:         0,
			lowerCategory: None,
			isPre:         false,
		}
		return &pvNext, nil
	case Alpha, Beta, Gamma, RC:
		return pv.lowerOK(vs)
	case None:
		return nil, errors.New("Did not specify how to upgrade the version")
	}
	return nil, errors.New("Did not specify how to upgrade the version")
}

var lowerMap = map[VersionSegment]map[VersionSegment]int{
	Alpha: {None: 1, Alpha: 1, Beta: 0, Gamma: 0, RC: 0},
	Beta:  {None: 1, Alpha: 2, Beta: 1, Gamma: 0, RC: 0},
	Gamma: {None: 1, Alpha: 2, Beta: 2, Gamma: 1, RC: 0},
	RC:    {None: 1, Alpha: 2, Beta: 2, Gamma: 2, RC: 1},
}

func (pv ParsedVersion) lowerOK(seg VersionSegment) (*ParsedVersion, error) {
	i := lowerMap[seg][pv.lowerCategory]
	if i == 0 {
		return nil, errors.New("Cannot create an " + seg.String() + " version if the current version is already a(n) " + pv.lowerCategory.String() + " one")
	}
	if i == 1 {
		pvNext := ParsedVersion{
			major:         pv.major,
			minor:         pv.minor,
			patch:         pv.patch,
			lower:         pv.lower + 1,
			lowerCategory: seg,
			isPre:         false,
		}
		return &pvNext, nil
	}
	if i == 2 {
		pvNext := ParsedVersion{
			major:         pv.major,
			minor:         pv.minor,
			patch:         pv.patch,
			lower:         1,
			lowerCategory: seg,
			isPre:         false,
		}
		return &pvNext, nil
	}
	panic("should not get here")
}

// ParsedVersionSlice is used to implement sort.Interface for a slice of *ParsedVersion
type ParsedVersionSlice []*ParsedVersion

func (s ParsedVersionSlice) Len() int { return len(s) }

func (s ParsedVersionSlice) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

func (s ParsedVersionSlice) Less(i, j int) bool {
	tvj := s[j]
	if tvj == nil {
		return false
	}

	tvi := s[i]
	if tvi == nil {
		return true
	}

	if tvi.major < tvj.major {
		return true
	}
	if tvi.major > tvj.major {
		return false
	}

	if tvi.minor < tvj.minor {
		return true
	}
	if tvi.minor > tvj.minor {
		return false
	}

	if tvi.patch < tvj.patch {
		return true
	}
	if tvi.patch > tvj.patch {
		return false
	}

	if tvi.lowerCategory < tvj.lowerCategory {
		return true
	}
	if tvi.lowerCategory > tvj.lowerCategory {
		return false
	}

	if tvi.lower < tvj.lower {
		return true
	}
	if tvi.lower > tvj.lower {
		return false
	}

	if tvi.isPre && !tvj.isPre {
		return true
	}
	return false
}
