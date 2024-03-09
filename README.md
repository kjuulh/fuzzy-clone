# fuzzy-clone

Fuzzy clone is a repository picker and downloader. It exists for one purpose, so that you don't have to fiddle around in a git ui (github, gitea, etc.), find a download link, cd on your local pc, git clone, cd again and so on.

Fuzzy clone simply presents a list of your subscribed orgs, and you fuzzy search, hit enter and you're placed in the repo, simply as that.

## Setup

Fuzzy clone either uses 

```bash
go install github.com/kjuulh/fuzzy-clone@latest

# Subscriptions
FUZZY_CLONE_GITHUB=true #default
FUZZY_CLONE_GITEA_ORGS=git.front.kjuulh.io/kjuulh

# Home
FUZZY_CLONE_ROOT=$HOME/git # default
# Will produce a structure like so once a repo is cloned
# $HOME/git/github.com/kjuulh/fuzzy-clone

# Authentication

FUZZY_CLONE_GITHUB_TOKEN=#<github token>
# Or fallbacks on
GITHUB_TOKEN=#<github token>
GITEA_TOKEN=#<gitea token>
```

## Prerequisites

Fuzzy-clone currently requires fzf and gh (github-cli) to function

## Usage

```
# Pick a repo
fuzzy-clone

# Update cache
fuzzy-clone cache update
```
