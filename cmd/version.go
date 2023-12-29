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
package cmd

import (
	"fmt"
	"regexp"
	"runtime/debug"
)

// From https://icinga.com/blog/2022/05/25/embedding-git-commit-information-in-go-binaries/,
// altered to the use case.

// commit is the commit hash
// uncommitted is whether there were other uncommitted changes when built
// FullCommit is the full commit string.
var commit, uncommitted, FullCommit = func() (string, bool, string) {
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
		return "unknown", false, " status - not built from repository"
	}
	return commit, modified, fmt.Sprint(commit, commitInfo)
}()

// version is the
var version = func() string { return "v0.0.0+pre" }()

var rxVersion *regexp.Regexp

func init() {
	rxVersion, _ = regexp.Compile(`v?(\d+)[.](\d+)[.](\d+)(?:-(alpha|beta|gamma|rc)[.](\d+))?(?:[+](pre))?`)
}

// FullVersion returns the version, including git status.
func FullVersion() string {
	v := version
	matches := rxVersion.FindStringSubmatch(v)
	if len(matches) > 0 && matches[6] == "pre" {
		v += fmt.Sprintf(" (snapshot: %s)", FullCommit)
	}
	return v
}

// Version returns the base version.
func Version() string {
	return version
}

// TODO
// struct ParsedVersion {}
// func GetParsedVersion() *ParsedVersion {}