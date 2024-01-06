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
	Long:                       `git next-tag: Update and commit the next tag/version of a git repository`,
	SilenceUsage:               true,
	PreRunE:                    func(cmd *cobra.Command, args []string) error { return initConfig() },
	RunE:                       func(cmd *cobra.Command, args []string) error { return nextTag(cmd, args) },
	SuggestionsMinimumDistance: 5,
}

// Execute runs the git-next-tag command
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
	b, err := askBoolean("Do you want to have an initial v on your versions?", true)
	if err != nil {
		return err
	}
	viper.Set("initial_v", b)

	b, err = askBoolean("Do you want to have annotated tags in your git repository?", false)
	if err != nil {
		return err
	}
	viper.Set("tag_annotated", b)

	b, err = askBoolean("Do you want to have any non-tagged version marked as a prerelease?", true)
	if err != nil {
		return err
	}
	viper.Set("always_leave_version_pre", b)

	// TODO: version_files
	// viper.Set("version_files", []string{})

	return nil
}

// askBoolean asks a boolean question.
func askBoolean(s string, def bool) (bool, error) {
	pos := 1
	if def {
		pos = 0
	}
	menu := promptui.Select{
		Label:     s,
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

		dryrun, _ := cmd.Flags().GetBool("dry-run")

		// TODO: save files, commit, tag, save files, commit.
		return doTagging(initialTag, head.Hash(), dryrun)
	}

	tagVersions := make([]*semver.ParsedVersion, len(tags))
	for k := range tags {
		pv := semver.ParseVersion(k)
		tagVersions = append(tagVersions, pv)
	}

	sort.Sort(semver.ParsedVersionSlice(tagVersions))

	// To turn the slice around so that the greatest versions are first
	slices.Reverse(tagVersions)

	pvCurrent := tagVersions[0]
	vCurrent := pvCurrent.String()
	// TODO: Add the v if required.
	slog.Debug("Current version: " + vCurrent)

	vs, err := getVersionSegment(cmd.Flags())
	if err != nil {
		return err
	}

	pvNext, err := pvCurrent.IncrementVersion(vs, false)
	if err != nil {
		return err
	}

	// TODO: Process files if any, then commit.

	out, err := exec.Command("git", "describe", "--contains", head.Hash().String()).Output()
	if err != nil {
		return err
	}

	possibleTag := string(out)
	if possibleTag != "" {
		return fmt.Errorf("Repository is already tagged with %s and no more commits have been made", possibleTag)
	}

	dryrun, _ := cmd.Flags().GetBool("dry-run")

	// TODO: Add the v if required.
	err = doTagging(pvNext.String(), head.Hash(), dryrun)
	if err != nil {
		return err
	}

	if viper.GetBool("always_leave_version_pre") {
		// TODO: if semver.Major or semver.Minor, make semver.Patch.
		pvFinal, err := pvNext.IncrementVersion(vs, true)
		if err != nil {
			return err
		}

		// TODO: Finish up here by changing files and committing, adding the v if required.
		return errors.New("Need to process files to add version " + pvFinal.String())
	}

	return nil
}

// askInitialTagging gets the initial tag version and asks whether to cancel.
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

// This is my 'TODO' area of functions I think I'll need soon.
//
//revive:disable:unused-parameter
func checkForNewVersion(fileName string, newVersion string) error {
	return errors.New("checkForNewVersion not implemented yet")
}
