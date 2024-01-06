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

	"github.com/csjewell/git-next-tag/semver"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var repo *git.Repository

var rootCmd = &cobra.Command{
	Use:                        "git-next-tag",
	Version:                    FullVersion(),
	Short:                      "Commit the next tag.",
	Long:                       `Update and commit the next tag/version of a git repository`,
	SilenceUsage:               true,
	PreRunE:                    func(cmd *cobra.Command, args []string) error { return initConfig() },
	RunE:                       nextTag,
	SuggestionsMinimumDistance: 5,
}

// Execute runs the git-next-tag command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
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
	storage, ok := repo.Storer.(*filesystem.Storage)
	if !ok {
		return errors.New("git not running on a filesystem?")
	}
	gitDir = path.Dir(storage.Filesystem().Root())

	// Search config in git root directory with name ".git-next-tag" (without extension).
	viper.AddConfigPath(gitDir)
	viper.SetConfigType("yaml")
	viper.SetConfigName(".git-next-tag")

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		slog.Debug("Using config file:" + viper.ConfigFileUsed())
	} else {
		// Initialize the file.
		err = askConfig()
		if err != nil {
			return errors.New("Configuration collection cancelled")
		}

		viper.Set("version_files", []string{})

		file := path.Join(gitDir, ".git-next-tag")
		err = viper.WriteConfigAs(file)
		if err != nil {
			return fmt.Errorf("Could not save configuration: %w", err)
		}
	}

	return nil
}

func askConfig() error {
	// TODO: Make this dependent on whether we have it set from a file.
	bAnswer, err := askBoolean("Do you want to have an initial v on your versions?", true)
	if err != nil {
		return err
	}
	viper.Set("initial_v", bAnswer)

	bAnswer, err = askBoolean("Do you want to have annotated tags in your git repository?", false)
	if err != nil {
		return err
	}
	viper.Set("tag_annotated", bAnswer)

	bAnswer, err = askBoolean("Do you want to have any non-tagged version marked as a prerelease?", true)
	if err != nil {
		return err
	}
	viper.Set("always_leave_version_pre", bAnswer)

	// TODO: version_files
	// viper.Set("version_files", []string{})

	return nil
}

// askBoolean asks a boolean question.
func askBoolean(sQuestion string, def bool) (bool, error) {
	pos := 1
	if def {
		pos = 0
	}
	menu := promptui.Select{
		Label:     sQuestion,
		CursorPos: pos,
		Items:     []string{"Yes", "No"},
	}

	_, ok, err := menu.Run()
	if err != nil {
		return false, errors.New("Cancelled")
	}
	if ok == "No" {
		return false, nil
	}
	return true, nil
}

// nextTag gets the next tag requested, saves it, etc.
func nextTag(cmd *cobra.Command, _ []string) error {
	err := isTreeClean()
	if err != nil {
		return err
	}

	tags, err := retrieveTags()
	if err != nil {
		return err
	}

	vsIncrement, pvNext, err := getNextVersion(cmd, tags)
	if err != nil {
		return err
	}

	vNext := normalizeVersion(pvNext.String())

	dryrun, _ := cmd.Flags().GetBool("dry-run")
	filesToProcess := viper.GetStringSlice("version_files")
	err = updateFiles(vNext, filesToProcess, dryrun)
	if err != nil {
		return err
	}

	head, err := checkAlreadyTagged()
	if err != nil {
		return err
	}

	err = doTagging(vNext, head.Hash(), dryrun)
	if err != nil {
		return err
	}

	if viper.GetBool("always_leave_version_pre") {
		err := afterTag(vsIncrement, pvNext, filesToProcess, dryrun)
		if err != nil {
			return err
		}
	}

	return nil
}

func afterTag(
	vsIncrement semver.VersionSegment,
	pvNext *semver.ParsedVersion,
	filesToProcess []string,
	dryrun bool,
) error {
	var vsNext semver.VersionSegment
	switch vsIncrement {
	case semver.Major, semver.Minor:
		vsNext = semver.Patch
	default:
		vsNext = vsIncrement
	}

	pvFinal, err := pvNext.IncrementVersion(vsNext, true)
	if err != nil {
		return err
	}
	vFinal := normalizeVersion(pvFinal.String())

	err = updateFiles(vFinal, filesToProcess, dryrun)
	if err != nil {
		return err
	}

	return nil
}

func checkAlreadyTagged() (*plumbing.Reference, error) {
	head, err := repo.Reference(plumbing.HEAD, true)
	if err != nil {
		return nil, err
	}

	headHash := head.Hash().String()
	out, err := exec.Command("git", "describe", "--contains", headHash).Output()
	if err != nil {
		return nil, err
	}

	possibleTag := string(out)
	if possibleTag != "" {
		return nil, fmt.Errorf("Repository is already tagged with %s and no more commits have been made", possibleTag)
	}

	return head, nil
}

// getNextVersion gets the next version based on previous one, if a previous one exists.
// Otherwise, the "next version" is 0.1.0.
func getNextVersion(cmd *cobra.Command, tags map[string]*object.Tag) (
	semver.VersionSegment, *semver.ParsedVersion, error,
) {
	var (
		vsIncrement semver.VersionSegment
		pvNext      *semver.ParsedVersion
		err         error
	)

	if len(tags) == 0 {
		vNext, err := askInitialTagging()
		if err != nil {
			return semver.NonSegment, nil, err
		}

		pvNext = semver.ParseVersion(vNext)
		vsIncrement = semver.Patch
	} else {
		tagVersions := make([]*semver.ParsedVersion, 0, len(tags))
		for k := range tags {
			pv := semver.ParseVersion(k)
			tagVersions = append(tagVersions, pv)
		}

		sort.Sort(semver.ParsedVersionSlice(tagVersions))

		// To turn the slice around so that the greatest versions are first
		slices.Reverse(tagVersions)

		pvCurrent := tagVersions[0]
		vCurrent := pvCurrent.String()
		slog.Debug("Current tag: " + normalizeVersion(vCurrent))

		vsIncrement, err = getVersionSegment(cmd.Flags())
		if err != nil {
			return semver.NonSegment, nil, err
		}

		pvNext, err = pvCurrent.IncrementVersion(vsIncrement, false)
		if err != nil {
			return semver.NonSegment, nil, err
		}
	}

	return vsIncrement, pvNext, err
}

func normalizeVersion(s string) string {
	initialV := viper.GetBool("initial_v")
	if initialV {
		return "v" + s
	}
	return s
}

// askInitialTagging gets the initial tag version and asks whether to cancel.
func askInitialTagging() (string, error) {
	versionInitial := normalizeVersion("0.1.0")

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
