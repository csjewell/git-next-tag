/*
Copyright © 2023, 2024 Curtis Jewell <golang@curtisjewell.name>
SPDX-License-Identifier: MIT

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

// Regexp is a regexp.Regexp that finds any possible version string within a line.
var Regexp = func() *regexp.Regexp {
	return regexp.MustCompile(`v?(\d+)[.](\d+)[.](\d+)(?:-(alpha|beta|gamma|rc)[.](\d+))?(?:[-](pre))?`)
}()

// RegexpString is a regexp.Regexp that validates a version string.
var RegexpString = func() *regexp.Regexp {
	return regexp.MustCompile(`\Av?(\d+)[.](\d+)[.](\d+)(?:-(alpha|beta|gamma|rc)[.](\d+))?(?:[-](pre))?\z`)
}()

// VersionSegment specifies what segment of the version is being changed.
type VersionSegment int

const (
	NonSegment VersionSegment = iota
	Major
	Minor
	Patch
	Alpha
	Beta
	Gamma
	RelCand
	Pre
)

var vsName = []string{"", "major", "minor", "patch", "alpha", "beta", "gamma", "rc", "pre"}

// String is provided in order for VersionSegment to satisfy the fmt.Stringer interface.
//
// This way, VersionSegment variables can be printed in fmt.Print and friends.
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
//
// It returns nil if the result is unparseable.
func ParseVersion(v string) *ParsedVersion {
	matches := RegexpString.FindStringSubmatch(v)
	if len(matches) == 0 {
		return nil
	}
	var pvAmswer ParsedVersion
	if strings.ToLower(matches[6]) == "pre" {
		pvAmswer.isPre = true
	}
	if strings.ToLower(matches[4]) == "alpha" {
		pvAmswer.lower, _ = strconv.Atoi(matches[5])
		pvAmswer.lowerCategory = Alpha
	}
	if strings.ToLower(matches[4]) == "beta" {
		pvAmswer.lower, _ = strconv.Atoi(matches[5])
		pvAmswer.lowerCategory = Beta
	}
	if strings.ToLower(matches[4]) == "gamma" {
		pvAmswer.lower, _ = strconv.Atoi(matches[5])
		pvAmswer.lowerCategory = Gamma
	}
	if strings.ToLower(matches[4]) == "rc" {
		pvAmswer.lower, _ = strconv.Atoi(matches[5])
		pvAmswer.lowerCategory = RelCand
	}
	pvAmswer.major, _ = strconv.Atoi(matches[1])
	pvAmswer.minor, _ = strconv.Atoi(matches[2])
	pvAmswer.patch, _ = strconv.Atoi(matches[3])

	return &pvAmswer
}

// String is provided in order to satisfy the fmt.Stringer interface.
//
// This way, ParsedVersion variables can be printed in fmt.Print and friends.
func (pv ParsedVersion) String() string {
	version := fmt.Sprint(pv.major, ".", pv.minor, ".", pv.patch)
	if pv.lowerCategory != NonSegment {
		version += fmt.Sprint("-", pv.lowerCategory, ".", pv.lower)
	}
	if pv.isPre {
		version += "-pre"
	}
	return version
}

// IncrementVersion ...
//
// If incrementing on the VersionSegment requested is impossible, an error is returned.
func (pv ParsedVersion) IncrementVersion(vsIncrement VersionSegment, isPre bool) (*ParsedVersion, error) {
	var pvNext ParsedVersion
	switch vsIncrement {
	case Major:
		pvNext = ParsedVersion{
			major:         pv.major + 1,
			minor:         0,
			patch:         0,
			lower:         0,
			lowerCategory: NonSegment,
			isPre:         isPre,
		}
		return &pvNext, nil
	case Minor:
		pvNext = ParsedVersion{
			major:         pv.major,
			minor:         pv.minor + 1,
			patch:         0,
			lower:         0,
			lowerCategory: NonSegment,
			isPre:         isPre,
		}
		return &pvNext, nil
	case Patch:
		pvNext = ParsedVersion{
			major:         pv.major,
			minor:         pv.minor,
			patch:         pv.patch + 1,
			lower:         0,
			lowerCategory: NonSegment,
			isPre:         isPre,
		}
		return &pvNext, nil
	case Alpha, Beta, Gamma, RelCand:
		return pv.lowerOK(vsIncrement, isPre)
	case Pre:
		if !pv.isPre {
			return nil, fmt.Errorf("Cannot upgrade non-prerelease version %s to non-prerelease", pv)
		}
		pvNext = ParsedVersion{
			major:         pv.major,
			minor:         pv.minor,
			patch:         pv.patch,
			lower:         pv.lower,
			lowerCategory: pv.lowerCategory,
			isPre:         isPre,
		}
		return &pvNext, nil
	case NonSegment:
		return nil, errors.New("Did not specify how to upgrade the version")
	}
	return nil, errors.New("Did not specify how to upgrade the version")
}

var lowerMap = map[VersionSegment]map[VersionSegment]int{
	Alpha:   {NonSegment: 1, Alpha: 1, Beta: 0, Gamma: 0, RelCand: 0},
	Beta:    {NonSegment: 1, Alpha: 2, Beta: 1, Gamma: 0, RelCand: 0},
	Gamma:   {NonSegment: 1, Alpha: 2, Beta: 2, Gamma: 1, RelCand: 0},
	RelCand: {NonSegment: 1, Alpha: 2, Beta: 2, Gamma: 2, RelCand: 1},
}

func (pv ParsedVersion) lowerOK(seg VersionSegment, isPre bool) (*ParsedVersion, error) {
	checkType := lowerMap[seg][pv.lowerCategory]
	if checkType == 0 {
		return nil, fmt.Errorf("Cannot create an %s version if the current version is already a(n) %s one",
			seg, pv.lowerCategory)
	}
	if checkType == 1 {
		pvNext := ParsedVersion{
			major:         pv.major,
			minor:         pv.minor,
			patch:         pv.patch,
			lower:         pv.lower + 1,
			lowerCategory: seg,
			isPre:         isPre,
		}
		return &pvNext, nil
	}
	if checkType == 2 {
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

// ParsedVersionSlice is defined in order to make slices of ParsedVersion sortable using sort.Sort
// (because ParsedVersionSlice implements sort.Interface.)
//
// Example:
//
//	pvs = []*semver.ParsedVersion{
//		semver.ParseVersion("0.2.0-pre"),
//		semver.ParseVersion("0.2.0"),
//		semver.ParseVersion("0.1.0"),
//	}
//
//	sort.Sort(ParsedVersionSlice(pvs))
//	// { semver.ParseVersion("0.1.0"), semver.ParseVersion("0.2.0-pre"), semver.ParseVersion("0.2.0") }
type ParsedVersionSlice []*ParsedVersion

func (s ParsedVersionSlice) Len() int { return len(s) }

func (s ParsedVersionSlice) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

func (s ParsedVersionSlice) Less(indexFirst, indexSecond int) bool {
	pvSecond := s[indexSecond]
	if pvSecond == nil {
		return false
	}

	pvFirst := s[indexFirst]
	if pvFirst == nil {
		return true
	}

	if pvFirst.major < pvSecond.major {
		return true
	}
	if pvFirst.major > pvSecond.major {
		return false
	}

	if pvFirst.minor < pvSecond.minor {
		return true
	}
	if pvFirst.minor > pvSecond.minor {
		return false
	}

	if pvFirst.patch < pvSecond.patch {
		return true
	}
	if pvFirst.patch > pvSecond.patch {
		return false
	}

	if pvFirst.lowerCategory == NonSegment && pvSecond.lowerCategory != NonSegment {
		return false
	}
	if pvFirst.lowerCategory != NonSegment && pvSecond.lowerCategory == NonSegment {
		return true
	}

	if pvFirst.lowerCategory < pvSecond.lowerCategory {
		return true
	}
	if pvFirst.lowerCategory > pvSecond.lowerCategory {
		return false
	}

	if pvFirst.lower < pvSecond.lower {
		return true
	}
	if pvFirst.lower > pvSecond.lower {
		return false
	}

	if pvFirst.isPre && !pvSecond.isPre {
		return true
	}
	if !pvFirst.isPre && pvSecond.isPre {
		return false
	}

	return false
}
