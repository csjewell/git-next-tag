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

	nextVersion, err := getSpecifiedVersion(cmd.Flags(), currentParsedVersion)
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
