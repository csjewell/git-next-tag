/*
Copyright © 2023, 2024 Curtis Jewell <golang@curtisjewell.name>

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

package cmd

import (
	"errors"
	"fmt"
	"regexp"
	"runtime/debug"

	"github.com/csjewell/git-next-tag/semver"
	"github.com/spf13/pflag"
)

// GetParsedVersion returnes the parsed version.
func GetParsedVersion() *semver.ParsedVersion { return semver.ParseVersion(Version) }

// From https://icinga.com/blog/2022/05/25/embedding-git-commit-information-in-go-binaries/,
// altered to the use case.

// commit is the commit hash
// uncommitted is whether there were other uncommitted or unknown changes when built
// fullCommit is the full commit string.
var commit, uncommitted, fullCommit = func() (string, bool, string) {
	commit := ""
	modified := false
	commitInfo := ""
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				commit = setting.Value
			case "vcs.modified":
				if setting.Value == "true" {
					modified = true
					commitInfo = " + uncommitted changes"
				}
			}
		}
	}
	if commit == "" {
		return "unknown", true, "unknown status - not built from repository"
	}
	return commit, modified, fmt.Sprint(commit, commitInfo)
}()

func init() {
	_ = FullVersion()
}

// FullVersion returns the version, including git status.
func FullVersion() string {
	v := Version
	rx, _ := regexp.Compile(`\-pre\z`)
	if rx.MatchString(v) {
		v += fmt.Sprintf(" (snapshot: %s)", fullCommit)
	}
	return v
}

func getVersionSegment(flags *pflag.FlagSet) (semver.VersionSegment, error) {
	getFlag := func(s string) bool {
		b, _ := flags.GetBool(s)
		return b
	}

	if getFlag("major") {
		return semver.Major, nil
	}
	if getFlag("minor") {
		return semver.Minor, nil
	}
	if getFlag("patch") {
		return semver.Patch, nil
	}
	if getFlag("alpha") {
		return semver.Alpha, nil
	}
	if getFlag("beta") {
		return semver.Beta, nil
	}
	if getFlag("gamma") {
		return semver.Gamma, nil
	}
	if getFlag("rc") {
		return semver.RC, nil
	}

	return semver.NonSegment, errors.New("Did not specify how to upgrade the version")
}
