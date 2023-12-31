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
	"log/slog"
	"os"
	"os/exec"
	"path"
	"slices"
	"sort"
	"strings"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// TODO: Check out using github.com/charmbracelet/bubbletea instead of promptui

var (
	cfgFile string
	repo    *git.Repository
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:                        "git next-tag",
	Version:                    FullVersion(),
	Short:                      "Commit the next tag.",
	Long:                       `...`,
	SilenceUsage:               true,
	PreRunE:                    func(cmd *cobra.Command, args []string) error { return initConfig() },
	RunE:                       func(cmd *cobra.Command, args []string) error { return nextTag(cmd, args) },
	SuggestionsMinimumDistance: 5,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.Flags().Bool("dry-run", true, "Do a dry-run only")
	rootCmd.Flags().Bool("major", false, "Increment major version")
	rootCmd.Flags().Bool("minor", false, "Increment minor version")
	rootCmd.Flags().Bool("patch", false, "Increment patch version")
	rootCmd.Flags().Bool("alpha", false, "Increment alpha version")
	rootCmd.Flags().Bool("beta", false, "Increment beta version")
	rootCmd.Flags().Bool("gamma", false, "Increment gamma version")
	rootCmd.Flags().Bool("rc", false, "Increment release candidate version")
}

// initConfig reads in and creates or updates a config file.
func initConfig() error {
	// Find current directory.
	gitDir, err := os.Getwd()
	if err != nil {
		return err
	}

	// Now get its top level git repository.
	// repo is a global, we will need it in nextTag.
	repo, err = git.PlainOpenWithOptions(gitDir, &git.PlainOpenOptions{
		DetectDotGit:          true,
		EnableDotGitCommonDir: true,
	})
	if err != nil {
		return err
	}

	// The return from Root() includes the .git directory, so shake it off.
	gitDir = path.Dir(repo.Storer.(*filesystem.Storage).Filesystem().Root())

	// Search config in git root directory with name ".git-next-tag" (without extension).
	viper.AddConfigPath(gitDir)
	viper.SetConfigType("yaml")
	viper.SetConfigName(".git-next-tag")

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		slog.Debug("Using config file:" + viper.ConfigFileUsed())
	} else {
		// Initialize the file.
		// TODO: Make this a 'question and answer' bit
		viper.Set("initial_v", true)
		viper.Set("tag_annotated", false)
		viper.Set("version_files", []string{})
		viper.Set("always_leave_version_pre", true)
		err = viper.WriteConfig()
		if err != nil {
			return fmt.Errorf("Could not save configuration: %w", err)
		}
	}

	return nil
}

func nextTag(cmd *cobra.Command, _ []string) error {
	err := isTreeClean()
	if err != nil {
		return err
	}

	head, err := repo.Reference(plumbing.HEAD, true)
	if err != nil {
		return err
	}

	tags, err := retrieveTags()
	if err != nil {
		return err
	}

	if len(tags) == 0 {
		initialTag, err := askInitialTagging()
		if err != nil {
			return err
		}
		return doTagging(initialTag, head.Hash())
	}

	tagNames := make([]string, 0, len(tags))
	for k := range tags {
		tagNames = append(tagNames, k)
	}

	tagVersions := make(map[string]*ParsedVersion)
	for _, s := range tagNames {
		pv := parseVersion(s)
		tagVersions[s] = pv
	}

	tagNames = sortTags(tagNames, tagVersions)

	currentVersion := tagNames[0]
	slog.Debug("Current version: " + currentVersion)

	currentParsedVersion := tagVersions[currentVersion]

	nextVersion, err := getSpecifiedVersion(cmd, currentParsedVersion)
	if err != nil {
		return err
	}

	// getRecommendedVersion(tags[currentVersion])
	out, err := exec.Command("git", "describe", "--contains", head.Hash().String()).Output()
	if err != nil {
		return err
	}
	possibleTag := string(out)
	if possibleTag != "" {
		return fmt.Errorf("Repository is already tagged with %s and no more commits have been made", possibleTag)
	}

	err = doTagging(nextVersion.String(), head.Hash())
	if err != nil {
		return err
	}

	/*

	   git push
	   git push --tags

	*/

	return nil
}

func getSpecifiedVersion(cmd *cobra.Command, pvCurrent *ParsedVersion) (*ParsedVersion, error) {
	flags := cmd.Flags()
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

func askInitialTagging() (string, error) {
	var versionInitial string
	initialV := viper.GetBool("initial_v")
	if initialV {
		versionInitial = "v0.1.0"
	} else {
		versionInitial = "0.1.0"
	}

	menu := promptui.Select{
		Label:     "Create initial tag to version " + versionInitial,
		CursorPos: 0,
		Items:     []string{"Yes", "No"},
	}

	_, ok, err := menu.Run()
	if err != nil {
		return "", errors.New("Cancelled initial tagging")
	}
	if ok == "No" {
		return "", errors.New("Cancelled initial tagging")
	}

	return versionInitial, nil
}

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

// This is my 'TODO' area of functions I think I'll need soon.
//
//revive:disable:unused-parameter
func checkForNewVersion(fileName string, newVersion string) error {
	return errors.New("checkForNewVersion not implemented yet")
}
