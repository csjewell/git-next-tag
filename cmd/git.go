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

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/manifoldco/promptui"
)

// TODO: Check out using github.com/charmbracelet/bubbletea instead of promptui

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
		menu := promptui.Select{
			Label:     "Git tree is not clean. Continue?",
			CursorPos: 0,
			Items:     []string{"Yes", "No"},
		}

		_, ok, err := menu.Run()
		if err != nil {
			return errors.New("Cancelled tagging because tree was not clean")
		}
		if ok == "No" {
			return errors.New("Cancelled tagging because tree was not clean")
		}
	}

	return nil
}

func retrieveTags() (map[string]*object.Tag, error) {
	tags := make(map[string]*object.Tag)

	// Start by checking if there are any tags
	iter, err := repo.Tags()
	if err != nil {
		return nil, err
	}

	if err := iter.ForEach(func(ref *plumbing.Reference) error {
		obj, err := repo.TagObject(ref.Hash())
		switch err {
		case nil:
			tags[obj.Name] = obj
		case plumbing.ErrObjectNotFound:
			// Not a tag object
		default:
			// Some other error
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return tags, nil
}

// This is my 'TODO' area of functions I think I'll need soon.
func doTagging(tag string, head plumbing.Hash) error {
	menu := promptui.Select{
		Label:     fmt.Sprintf("Creating tag for version %s. Continue?", tag),
		CursorPos: 0,
		Items:     []string{"Yes", "No"},
	}

	_, ok, err := menu.Run()
	if err != nil {
		return errors.New("Tagging cancelled")
	}
	if ok == "No" {
		return errors.New("Tagging cancelled")
	}

	_, err = repo.CreateTag(tag, head, nil)
	if err != nil {
		return err
	}
	slog.Info("Tagged revision %s as version %s", head.String(), tag)

	/*

	   git push
	   git push --tags

	*/

	return errors.New("Tagging not implemented yet")
}

//revive:disable:unused-parameter
func getRecommendedVersion(tag *object.Tag) error {
	// headRef, err := repo.Reference(plumbing.HEAD, true)

	// initialCommit, err := tag.Commit()

	return fmt.Errorf("getRecommendedVersion not implemented yet")
}
