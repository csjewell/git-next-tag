/*
Copyright © 2024 Curtis Jewell <golang@curtisjewell.name>

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

package semver_test

import (
	"sort"
	"testing"

	"github.com/csjewell/git-next-tag/semver"
	"github.com/google/go-cmp/cmp"
)

func TestParseVersion(t *testing.T) {
	pv := semver.ParseVersion("")
	if pv != nil {
		t.Error("incorrect result: expected nil, got", pv)
	}

	pv = semver.ParseVersion("1.0")
	if pv != nil {
		t.Error("incorrect result: expected nil, got", pv)
	}

	pv = semver.ParseVersion("1.0.0")
	if pv == nil {
		t.Error("incorrect result: Could not parse 1.0.0")
	}
	if pv.String() != "1.0.0" {
		t.Error("incorrect result: expected 1.0.0-pre, got", pv)
	}

	pv = semver.ParseVersion("1.0.0-pre")
	if pv == nil {
		t.Error("incorrect result: Could not parse 1.0.0-pre")
	}
	if pv.String() != "1.0.0-pre" {
		t.Error("incorrect result: expected 1.0.0-pre, got", pv)
	}

	pv = semver.ParseVersion("1.0.0-pre.2")
	if pv != nil {
		t.Error("incorrect result: expected nil, got", pv)
	}

	pv = semver.ParseVersion("v1.0.0-alpha.2")
	if pv == nil {
		t.Error("incorrect result: Could not parse 1.0.0-alpha.2")
	}
	if pv.String() != "1.0.0-alpha.2" {
		t.Error("incorrect result: expected 1.0.0-alpha.2, got", pv)
	}

	pv = semver.ParseVersion("1.0.0-alpha.2-beta.1")
	if pv != nil {
		t.Error("incorrect result: expected nil, got", pv)
	}

	pv = semver.ParseVersion("1.2.3-alpha.2-pre")
	if pv == nil {
		t.Error("incorrect result: Could not parse 1.1.1-alpha.2-pre")
	}
	if pv.String() != "1.2.3-alpha.2-pre" {
		t.Error("incorrect result: expected 1.2.3-alpha.2-pre, got", pv)
	}
}

func TestIncrementVersion(t *testing.T) {
	pv := semver.ParseVersion("1.2.3-beta.2-pre")
	var (
		pvResp *semver.ParsedVersion
		err    error
	)

	_, err = pv.IncrementVersion(semver.NonSegment, false)
	if err == nil {
		t.Error("Did not get error when expected")
	}

	_, err = pv.IncrementVersion(semver.Alpha, false)
	if err == nil {
		t.Error("Did not get error when expected")
	}

	pvResp, err = pv.IncrementVersion(semver.Beta, false)
	if err != nil {
		t.Error("Got error", err)
	}
	if pvResp.String() != "1.2.3-beta.3" {
		t.Error("Did not get 1.2.3-beta.3, got", pvResp)
	}

	pvResp, err = pv.IncrementVersion(semver.Gamma, false)
	if err != nil {
		t.Error("Got error", err)
	}
	if pvResp.String() != "1.2.3-gamma.1" {
		t.Error("Did not get 1.2.3-gamma.1, got", pvResp)
	}

	pvResp, err = pv.IncrementVersion(semver.RC, false)
	if err != nil {
		t.Error("Got error", err)
	}
	if pvResp.String() != "1.2.3-rc.1" {
		t.Error("Did not get 1.2.3-rc.1, got", pvResp)
	}

	pvResp, err = pv.IncrementVersion(semver.Patch, false)
	if err != nil {
		t.Error("Got error", err)
	}
	if pvResp.String() != "1.2.4" {
		t.Error("Did not get 1.2.4, got", pvResp)
	}

	pvResp, err = pv.IncrementVersion(semver.Minor, false)
	if err != nil {
		t.Error("Got error", err)
	}
	if pvResp.String() != "1.3.0" {
		t.Error("Did not get 1.3.0, got", pvResp)
	}

	pvResp, err = pv.IncrementVersion(semver.Major, false)
	if err != nil {
		t.Error("Got error", err)
	}
	if pvResp.String() != "2.0.0" {
		t.Error("Did not get 2.0.0, got", pvResp)
	}
}

func TestSorting(t *testing.T) {
	versions := []string{
		"0.2.0",
		"0.1.0",
		"0.1.0-pre",
		"0.1.0-gamma.1-pre",
		"0.1.0-alpha.1",
		"0.1.0-beta.1",
		"3.3.1",
		"0.1.0-gamma.1",
		"0.1.1",
		"3.0.1",
	}

	expected := []*semver.ParsedVersion{
		semver.ParseVersion("0.1.0-alpha.1"),
		semver.ParseVersion("0.1.0-beta.1"),
		semver.ParseVersion("0.1.0-gamma.1-pre"),
		semver.ParseVersion("0.1.0-gamma.1"),
		semver.ParseVersion("0.1.0-pre"),
		semver.ParseVersion("0.1.0"),
		semver.ParseVersion("0.1.1"),
		semver.ParseVersion("0.2.0"),
		semver.ParseVersion("3.0.1"),
		semver.ParseVersion("3.3.1"),
	}

	versionsParsed := make([]*semver.ParsedVersion, 0, 10)
	for _, v := range versions {
		versionsParsed = append(versionsParsed, semver.ParseVersion(v))
	}

	sort.Sort(semver.ParsedVersionSlice(versionsParsed))

	o := cmp.Comparer(func(x, y *semver.ParsedVersion) bool { return x.String() == y.String() })
	if diff := cmp.Diff(expected, versionsParsed, o); diff != "" {
		t.Error(diff)
	}
}
