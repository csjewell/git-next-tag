# git-next-tag

(Note: This is aspirational as yet, not every capability described here has been implemented yet.
Enhancements are welcomed.)

## Usage

    git next-tag --major|minor|patch|alpha|beta|gamma|rc [--dry-run]

git-next-tag (the first dash in the name is optional) will read the highest current version, 
increment the requested segment of the tag, write to any files where it is told to update the version, commit those changes to git, create and push a tag, and if desired, make a commit setting a pre-release for the next version.

One of the segment tags is required at this point. (It is a TODO to determine what to increment using [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/) tags in the commits.)

## Version format:

The versions used by `git next-tag` are a subset of the ones described in [Semantic Versioning 2.0.0](https://semver.org/spec/v2.0.0.html). They can begin with a v or not, and must have major, minor, and patch levels. After the patch level, they can have an optional release-state-modifier of `-alpha.#`, `-beta.#`, `-gamma.#`, or `-rc.#`, where # increases from 1. In addition, versions can have a "pre-release marker" of `-pre`, indicating that the version is a prerelease of what is otherwise specified.

Examples:

    v1.0.1
    2.5.0
    0.2.1-alpha.1
    v0.2.1-pre
    1.4.9-rc.16-pre

By default, the +pre marker, while not used in tags, will be used in any files where git-next-tag is allowed to update the version, AFTER a new tag is committed. After the new tag is committed, the last number in the version (either the patch level or the release-state modifier) will be incremented, and the -pre will be added. Then a new automatically-generated commit will be pushed. This behavior can be turned off.

## Configuration file

The configuration file is a YAML file named .git-next-tag within the root of the git repository.
