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
	"regexp"
	"strconv"
	"strings"
)

var rxVersion *regexp.Regexp = func() *regexp.Regexp {
	rx, _ := regexp.Compile(`v?(\d+)[.](\d+)[.](\d+)(?:-(alpha|beta|gamma|rc)[.](\d+))?(?:[+](pre))?`)
	return rx
}()

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
