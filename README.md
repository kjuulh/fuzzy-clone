# fuzzy-clone (fz)

![fuzzy-clone demo](assets/fuzzy-clone.gif)

> <small><strong>,</strong> is my alias for <strong>fz</strong></small>

Fuzzy clone is a repository picker and downloader. It exists for one purpose, so that you don't have to fiddle around in a git ui (github, gitea, etc.), find a download link, cd on your local pc, git clone, cd again and so on.

Fuzzy clone simply presents a list of your subscribed orgs, and you fuzzy search, hit enter and you're placed in the repo, simply as that.

## Install

### Golang

```bash
go install github.com/kjuulh/fuzzy-clone@latest
```

### Homebrew

```bash
brew install kjuulh/brew/fuzzy-clone  

fz
```

## Setup

Setup the utility scripts

### Zsh

```bash
echo 'eval "$(fuzzy-clone init zsh)" >> ~/.zshrc' 

fz
```

## Usage

```
# Pick a repo
fuzzy-clone

# Update cache (that way fuzzy-clone will be next to instant. 
fuzzy-clone cache update
```

## Configuration
Fuzzy clone either uses 

```bash
# Home
FUZZY_CLONE_ROOT=$HOME/git # default
# Will produce a structure like so once a repo is cloned
# $HOME/git/github.com/kjuulh/fuzzy-clone

# Authentication
FUZZY_CLONE_GITHUB_TOKEN=#<github token>
# Or fallbacks on
GITHUB_ACCESS_TOKEN=#<github token>
```
