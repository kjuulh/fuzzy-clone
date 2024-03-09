# fuzzy-clone

Fuzzy clone is a repository picker and downloader. It exists for one purpose, so that you don't have to fiddle around in a git ui (github, gitea, etc.), find a download link, cd on your local pc, git clone, cd again and so on.

Fuzzy clone simply presents a list of your subscribed orgs, and you fuzzy search, hit enter and you're placed in the repo, simply as that.

## Setup

Fuzzy clone either uses 

```bash
go install github.com/kjuulh/fuzzy-clone@latest

# Home
FUZZY_CLONE_ROOT=$HOME/git # default
# Will produce a structure like so once a repo is cloned
# $HOME/git/github.com/kjuulh/fuzzy-clone

# Authentication
FUZZY_CLONE_GITHUB_TOKEN=#<github token>
# Or fallbacks on
GITHUB_ACCESS_TOKEN=#<github token>
```

## Usage

```
# Pick a repo
fuzzy-clone

# Update cache (that way fuzzy-clone will be next to instant. 
fuzzy-clone cache update
```

## Script

To make automatic hopping possible you need to use one of the shell variants, see `shell/zsh/fuzzy-clone.zsh`.

Simply create an alias in your plugin or .zshrc

Remember fuzzy-clone.zsh has to be moved to your path to be useful

`alias fc="source fuzzy-clone.zsh"`
