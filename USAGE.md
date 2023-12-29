# Using git-next-tag

The first thing to do is install it. You can do that one of two ways:

1. `go install github.com/csjewell/git-next-tag`
2. Go to https://github.com/csjewell/git-next-tag/releases and download the newest release for your operating system, unpack it, and put it somewhere in your path.

Once you have done that, run `git next-tag` once at the top level of any of your repositories. It'll create a YAML file for you named .git-next-tag containing the settings. If you already have that file, it will make sure it has all the current settings, asking you any new questions that need asked.

You can do these two things at any time to upgrade git-next-tag. No other requirements necessary.

Once you have done that, `git add .git-next-tag; git commit -m "chore: Adding git-next-tag support"` and then `git next-tag --patch` will push up a tag with the next version for you.

To find out more usage information, just run `git next-tag --help` and it'll tell you what you need.
