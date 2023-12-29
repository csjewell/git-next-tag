# Contributing to git-next-tag

## Committing

individual commits should have the [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) tags. In my case, to help me do so, I've installed [gbc](https://github.com/AllanCapistrano/gbc) to do that. (I may write a better helper next.)

One goal of this program is to autosuggest which version to go to using the tags. I'd like to implement that before v1.0.0.

Note that the CI action on github also runs [golangci-lint](https://golangci-lint.run/) with [revive](https://revive.run/) as a linter. Please make sure that everything marked as an error is fixed. Warnings are fine to leave in temporarily. 

## Releasing

When I'm ready to release a version, I'll tag it with itself. Once that tag is pushed to GitHub, I have a [GitHub action](https://github.com/csjewell/git-next-tag/tree/dev/.github/workflows/goreleaser.yml) that uses [goreleaser](https://goreleaser.com/) to perform the release process.

Releases will be seen at https://github.com/csjewell/git-next-tag/releases a few minutes after being tagged.
