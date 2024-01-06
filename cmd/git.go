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

package cmd

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/manifoldco/promptui"
)

// isTreeClean checks if the tree is clean and cancels if requested.
func isTreeClean() error {
	wt, err := repo.Worktree()
	if err != nil {
		return err
	}

	status, err := wt.Status()
	if err != nil {
		return err
	}

	if !status.IsClean() {
		prompt := promptui.Prompt{
			Label:     "Git tree is not clean. Continue?",
			IsConfirm: true,
		}

		_, err := prompt.Run()
		if err != nil {
			return errors.New("Cancelled tagging because tree was not clean")
		}
	}

	return nil
}

// retrieveTags retrieves all tags in the current repository.
func retrieveTags() (map[string]*object.Tag, error) {
	tags := make(map[string]*object.Tag)

	// Start by checking if there are any tags
	iter, err := repo.Tags()
	if err != nil {
		return nil, err
	}

	if err := iter.ForEach(func(ref *plumbing.Reference) error {
		obj, err := repo.TagObject(ref.Hash())
		if err != nil {
			return err
		}
		tags[obj.Name] = obj
		return nil
	}); err != nil {
		return nil, err
	}

	return tags, nil
}

// doTagging applies a new tag to the repository and pushes to all remotes.
func doTagging(tag string, head plumbing.Hash, dryrun bool) error {
	prompt := promptui.Prompt{
		Label:     fmt.Sprintf("Creating tag for version %s. Continue?", tag),
		IsConfirm: true,
	}

	_, err := prompt.Run()
	if err != nil {
		return errors.New("Tagging cancelled")
	}

	if dryrun {
		return errors.New("Tagging cancelled due to dry-run")
	}

	_, err = repo.CreateTag(tag, head, nil)
	if err != nil {
		return err
	}
	slog.Info(fmt.Sprintf("Tagged revision %s as version %s", head.String(), tag))

	remotes, err := repo.Remotes()
	if err != nil {
		return err
	}

	for _, remote := range remotes {
		slog.Debug(fmt.Sprintf("Pushing to %s", remote.Config().Name))
		err = remote.Push(&git.PushOptions{
			RemoteName: remote.Config().Name,
			RefSpecs:   remote.Config().Fetch,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func updateFiles(version string, filesToProcess []string, dryrun bool) error {
	if dryrun {
		return nil
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return err
	}

	for _, file := range filesToProcess {
		err = replaceInFile(file, version)
		if err != nil {
			return err
		}

		_, err = worktree.Add(file)
		if err != nil {
			return err
		}
	}

	if len(filesToProcess) != 0 {
		_, err = worktree.Commit("chore: Updating version to "+version, &git.CommitOptions{
			Amend:             false,
			All:               false,
			AllowEmptyCommits: false,
			Author:            nil,
			Committer:         nil,
			SignKey:           nil,
			Parents:           []plumbing.Hash{},
		})
		if err != nil {
			return err
		}
	}

	return nil
}
