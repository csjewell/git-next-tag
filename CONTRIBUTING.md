# Contributing to git-next-tag

## Committing

individual commits should have the [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) tags. In my case, to help me do so, I've installed [gbc](https://github.com/AllanCapistrano/gbc) to do that. (I may write a better helper next.)

One goal of this program is to autosuggest which version to go to using the tags. I'd like to implement that before v1.0.0.

## Releasing

When I'm ready to release a version, I'll tag it with itself. Once that tag is pushed to GitHub, I have a [GitHub action](https://github.com/csjewell/git-next-tag/tree/dev/.github/workflows/goreleaser.yml) that uses [goreleaser](https://goreleaser.com/) to perform the release process.

Releases will be seen at https://github.com/csjewell/git-next-tag/releases a few minutes after being tagged.
