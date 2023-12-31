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

// Package cmd implements the CLI structure of git-next-tag.
package cmd

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/pflag"
)

var rxVersion = func() *regexp.Regexp {
	rx, _ := regexp.Compile(`v?(\d+)[.](\d+)[.](\d+)(?:-(alpha|beta|gamma|rc)[.](\d+))?(?:[+](pre))?`)
	return rx
}()

// VersionSegment specifies what segment of the version is being changed.
type VersionSegment int

const (
	versionNone VersionSegment = iota
	versionMajor
	versionMinor
	versionPatch
	versionAlpha
	versionBeta
	versionGamma
	versionRC
	versionPre
)

var vsName = []string{"", "major", "minor", "patch", "alpha", "beta", "gamma", "rc", "pre"}

func (vs VersionSegment) String() string {
	return vsName[vs]
}

// TODO
type ParsedVersion struct {
	major         int
	minor         int
	patch         int
	lower         int
	lowerCategory VersionSegment
	isPre         bool
}

func GetParsedVersion() *ParsedVersion { return parseVersion(Version) }

func parseVersion(v string) *ParsedVersion {
	matches := rxVersion.FindStringSubmatch(v)
	if len(matches) == 0 {
		return nil
	}
	var pv ParsedVersion
	if strings.ToLower(matches[6]) == "pre" {
		pv.isPre = true
	}
	if strings.ToLower(matches[4]) == "alpha" {
		pv.lower, _ = strconv.Atoi(matches[5])
		pv.lowerCategory = versionAlpha
	}
	if strings.ToLower(matches[4]) == "beta" {
		pv.lower, _ = strconv.Atoi(matches[5])
		pv.lowerCategory = versionBeta
	}
	if strings.ToLower(matches[4]) == "gamma" {
		pv.lower, _ = strconv.Atoi(matches[5])
		pv.lowerCategory = versionGamma
	}
	if strings.ToLower(matches[4]) == "rc" {
		pv.lower, _ = strconv.Atoi(matches[5])
		pv.lowerCategory = versionRC
	}
	pv.major, _ = strconv.Atoi(matches[1])
	pv.minor, _ = strconv.Atoi(matches[2])
	pv.patch, _ = strconv.Atoi(matches[3])

	return &pv
}

func (pv *ParsedVersion) String() string {
	s := fmt.Sprint(pv.major, ".", pv.minor, ".", pv.patch)
	if pv.lowerCategory != versionNone {
		s = fmt.Sprint(s, "-", pv.lowerCategory, ".", pv.lower)
	}
	if pv.isPre {
		s += "+pre"
	}
	return s
}

func getSpecifiedVersion(flags *pflag.FlagSet, pvCurrent *ParsedVersion) (*ParsedVersion, error) {
	getFlag := func(s string) bool {
		b, _ := flags.GetBool(s)
		return b
	}
	var pv ParsedVersion
	if getFlag("major") {
		pv = ParsedVersion{
			major:         pvCurrent.major + 1,
			minor:         0,
			patch:         0,
			lower:         0,
			lowerCategory: versionNone,
			isPre:         false,
		}
		return &pv, nil
	}
	if getFlag("minor") {
		pv = ParsedVersion{
			major:         pvCurrent.major,
			minor:         pvCurrent.minor + 1,
			patch:         0,
			lower:         0,
			lowerCategory: versionNone,
			isPre:         false,
		}
		return &pv, nil
	}
	if getFlag("patch") {
		pv = ParsedVersion{
			major:         pvCurrent.major,
			minor:         pvCurrent.minor,
			patch:         pvCurrent.patch + 1,
			lower:         0,
			lowerCategory: versionNone,
			isPre:         false,
		}
		return &pv, nil
	}
	if getFlag("alpha") {
		return lowerOK(pvCurrent, versionAlpha)
	}
	if getFlag("beta") {
		return lowerOK(pvCurrent, versionBeta)
	}
	if getFlag("gamma") {
		return lowerOK(pvCurrent, versionGamma)
	}
	if getFlag("rc") {
		return lowerOK(pvCurrent, versionGamma)
	}

	return nil, errors.New("Did not specify how to upgrade the version")
}

var lowerMap = map[VersionSegment]map[VersionSegment]int{
	versionAlpha: {versionNone: 1, versionAlpha: 1, versionBeta: 0, versionGamma: 0, versionRC: 0},
	versionBeta:  {versionNone: 1, versionAlpha: 2, versionBeta: 1, versionGamma: 0, versionRC: 0},
	versionGamma: {versionNone: 1, versionAlpha: 2, versionBeta: 2, versionGamma: 1, versionRC: 0},
	versionRC:    {versionNone: 1, versionAlpha: 2, versionBeta: 2, versionGamma: 2, versionRC: 1},
}

func lowerOK(pvCurrent *ParsedVersion, seg VersionSegment) (*ParsedVersion, error) {
	i := lowerMap[seg][pvCurrent.lowerCategory]
	if i == 0 {
		return nil, errors.New("Cannot create an " + seg.String() + " version if the current version is already a(n) " + pvCurrent.lowerCategory.String() + " one")
	}
	if i == 1 {
		pv := ParsedVersion{
			major:         pvCurrent.major,
			minor:         pvCurrent.minor,
			patch:         pvCurrent.patch,
			lower:         pvCurrent.lower + 1,
			lowerCategory: seg,
			isPre:         false,
		}
		return &pv, nil
	}
	if i == 2 {
		pv := ParsedVersion{
			major:         pvCurrent.major,
			minor:         pvCurrent.minor,
			patch:         pvCurrent.patch,
			lower:         1,
			lowerCategory: seg,
			isPre:         false,
		}
		return &pv, nil
	}
	panic("should not get here")
}

func sortTags(tagNames []string, tagVersions map[string]*ParsedVersion) []string {
	sort.Slice(tagNames, func(i, j int) bool {
		tvj := tagVersions[tagNames[j]]
		if tvj == nil {
			return false
		}

		tvi := tagVersions[tagNames[i]]
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
	})

	// To turn the slice around so that the greatest versions are first
	slices.Reverse(tagNames)
	return tagNames
}
