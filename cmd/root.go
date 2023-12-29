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
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type VersionSegment int

const (
	versionMajor VersionSegment = iota
	versionMinor
	versionPatch
	versionAlpha
	versionBeta
	versionGamma
	versionRC
	versionPre
)

var (
	cfgFile string
	repo    *git.Repository
	segment *VersionSegment
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "git next-tag",
	Version: FullVersion(),
	Short:   "Commit the next tag.",
	Long:    `...`,
	PreRunE: func(cmd *cobra.Command, args []string) error { return initConfig() },
	RunE:    func(cmd *cobra.Command, args []string) error { return nextTag(cmd, args) },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	fmt.Println(Version())
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.Flags().Bool("major", false, "Increment major version")
	rootCmd.Flags().Bool("minor", false, "Increment minor version")
	rootCmd.Flags().Bool("patch", false, "Increment patch version")
}

// initConfig reads in config file and ENV variables if set.
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
		viper.Set("tag_annotated", true)
		viper.Set("version_files", []string{})
		viper.Set("always_leave_version_pre", true)
		viper.WriteConfig()
	}

	return nil
}

func nextTag(cmd *cobra.Command, args []string) error {
	// Start by checking for a clean tree.
	wt, err := repo.Worktree()
	if err != nil {
		return err
	}
	status, err := wt.Status()
	if err != nil {
		return err
	}
	if !status.IsClean() {
		return errors.New("Git tree is not clean.")
	}

	tags := make(map[string]*object.Tag)
	tagNames := make([]string, 100)

	// Start by checking if there are any tags
	iter, err := repo.Tags()
	if err != nil {
		return err
	}

	if err := iter.ForEach(func(ref *plumbing.Reference) error {
		obj, err := repo.TagObject(ref.Hash())
		switch err {
		case nil:
			tags[obj.Name] = obj
			tagNames = append(tagNames, obj.Name)
		case plumbing.ErrObjectNotFound:
			// Not a tag object
		default:
			// Some other error
			return err
		}
		return nil
	}); err != nil {
		return err
	}

	// sort.Slice()

	/*

	   VERSION=""

	   #get parameters
	   while getopts v: flag
	   do
	     case "${flag}" in
	       v) VERSION=${OPTARG};;
	     esac
	   done

	   #get highest tag number, and add 1.0.0 if doesn't exist
	   CURRENT_VERSION=`git describe --abbrev=0 --tags 2>/dev/null`

	   if [[ $CURRENT_VERSION == '' ]]
	   then
	     CURRENT_VERSION='v1.0.0'
	   fi
	   echo "Current Version: $CURRENT_VERSION"

	   # Get rid of the v by splitting it off.
	   CURRENT_VERSION_PARTS=(${CURRENT_VERSION//v/ })
	   VNUM=${CURRENT_VERSION_PARTS[0]}

	   #replace . with space so can split into an array
	   CURRENT_VERSION_PARTS=(${VNUM//./ })

	   #get number parts
	   VNUM1=${CURRENT_VERSION_PARTS[0]}
	   VNUM2=${CURRENT_VERSION_PARTS[1]}
	   VNUM3=${CURRENT_VERSION_PARTS[2]}

	   if [[ $VERSION == 'major' ]]
	   then
	     VNUM1=$((VNUM1+1))
	     VNUM2=0
	     VNUM3=0
	   elif [[ $VERSION == 'minor' ]]
	   then
	     VNUM2=$((VNUM2+1))
	     VNUM3=0
	   elif [[ $VERSION == 'patch' ]]
	   then
	     VNUM3=$((VNUM3+1))
	   else
	     echo "No version type (https://semver.org/) or incorrect type specified, try: -v [major, minor, patch]"
	     exit 1
	   fi

	   #create new tag, putting back the v.
	   NEW_TAG="v$VNUM1.$VNUM2.$VNUM3"
	   echo "($VERSION) updating $CURRENT_VERSION to $NEW_TAG"

	   #get current hash and see if it already has a tag
	   GIT_COMMIT=`git rev-parse HEAD`
	   NEEDS_TAG=`git describe --tags --contains $GIT_COMMIT 2>/dev/null`

	   #only tag if no tag already
	   if [ -z "$NEEDS_TAG" ]; then
	     git stash
	     git tag $NEW_TAG
	     echo "Tagged with $NEW_TAG"
	     git push
	     git push --tags
	   else
	     echo "Already a tag on this commit"
	   fi

	   exit 0


	*/

	return nil
}
