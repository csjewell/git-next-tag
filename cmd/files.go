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
	"fmt"
	"os"
	"slices"
	"strings"
)

func replaceInFile(fileName, newVersion string) error {
	input, err := os.ReadFile(fileName)
	if err != nil {
		return fmt.Errorf("Could not read file %s: %w", fileName, err)
	}

	fi, err := os.Stat(fileName)
	if err != nil {
		return fmt.Errorf("Could not get information about file %s: %w", fileName, err)
	}

	lines := strings.Split(string(input), "\n")

	for i, line := range lines {
		finds := rxVersion.FindAllString(line, -1)
		finds = slices.Compact(finds)
		if len(finds) != 0 {
			for _, find := range finds {
				if find != newVersion {
					line = strings.Replace(line, find, newVersion, -1)
				}
			}

			lines[i] = line
		}
	}

	output := strings.Join(lines, "\n")
	err = os.WriteFile(fileName, []byte(output), fi.Mode().Perm())
	if err != nil {
		return fmt.Errorf("Could not write to file %s: %w", fileName, err)
	}

	return nil
}

func createVersionDotGoFile(pkg, fileName string) error {
	versionFile := `
package ` + pkg + `

// Version is the current version of the library or command
var Version = func() string { return "v0.1.0" }()
`

	//revive:disable:add-constant
	err := os.WriteFile(fileName, []byte(versionFile), 0644)
	//revive:enable:add-constant
	if err != nil {
		return fmt.Errorf("Could not write to file %s: %w", fileName, err)
	}

	return nil
}
