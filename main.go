package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"

	"github.com/adrg/xdg"
	"github.com/google/go-github/v60/github"
	"github.com/ktr0731/go-fuzzyfinder"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

type Repository interface {
	Provider() string
	GetOrClone(ctx context.Context, root string) (string, error)
	ToString() string
}

func toRepos[T Repository](items []T) []Repository {
	repos := make([]Repository, 0, len(items))
	for _, item := range items {
		repos = append(repos, item)
	}

	return repos
}

type GitHubRepository struct {
	FullName string  `json:"fullName"`
	SshUrl   *string `json:"sshUrl"`
	HttpsUrl *string `json:"httpsUrl"`
}

func (g *GitHubRepository) ToString() string {
	return path.Join(g.Provider(), g.FullName)
}

func (*GitHubRepository) Provider() string {
	return "github.com"
}

func (g *GitHubRepository) GetOrClone(ctx context.Context, root string) (string, error) {
	destDir := path.Join("$HOME/git", g.Provider(), g.FullName)
	destDir = os.ExpandEnv(destDir)

	if _, err := os.Stat(destDir); errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(destDir, 0o755); err != nil {
			return "", fmt.Errorf("failed to prepare git dir: %w", err)
		}
	}

	entries, err := os.ReadDir(destDir)
	if err != nil {
		return "", fmt.Errorf("failed to read %s, %w", destDir, err)
	}

	if len(entries) == 0 {
		tryClone := func(url string) error {
			process := exec.Command("git", "clone", url, ".")
			process.Dir = destDir

			if err := process.Start(); err != nil {
				return err
			}

			log.Println("Downloading...")

			if err := process.Wait(); err != nil {
				return err
			}

			return nil
		}

		if g.SshUrl != nil {
			if err := tryClone(*g.SshUrl); err == nil {
				return destDir, nil
			}

			log.Printf("failed to clone with ssh, falling back to http: %s", err)
		}

		if g.HttpsUrl != nil {
			if err := tryClone(*g.HttpsUrl); err == nil {
				return destDir, nil
			}

			return "", fmt.Errorf("failed to clone with http: %w", err)
		}

		return "", fmt.Errorf("failed to clone repository: %s", g.FullName)
	}

	return destDir, nil
}

var _ Repository = &GitHubRepository{}

func NewGitHubRepository(repo *github.Repository) *GitHubRepository {
	return &GitHubRepository{
		FullName: repo.GetFullName(),
		SshUrl:   repo.SSHURL,
		HttpsUrl: repo.CloneURL,
	}
}

const Prefix = "fuzzy-clone"

type Cache struct {
	location string
	fileName string
}

func NewCache() *Cache {
	return &Cache{
		location: path.Join(xdg.CacheHome, Prefix, "cache"),
		fileName: "cache.json",
	}
}

type cachedRepositories struct {
	GitHub []*GitHubRepository `json:"github.com"`
}

func (c *cachedRepositories) ToRepos() []Repository {
	repos := make([]Repository, 0)

	for _, repo := range c.GitHub {
		repos = append(repos, repo)
	}

	return repos
}

func (c *Cache) Get(ctx context.Context) ([]Repository, bool, error) {
	contents, err := os.ReadFile(c.path())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, nil
		}

		return nil, false, fmt.Errorf("failed to read cache file: %w", err)
	}

	var groupedRepos cachedRepositories
	if err := json.Unmarshal(contents, &groupedRepos); err != nil {
		return nil, false, fmt.Errorf("failed to unmarshal: %s, %w", c.path(), err)
	}

	repos := groupedRepos.ToRepos()

	if len(repos) <= 0 {
		return nil, false, nil
	}

	return repos, true, nil
}

func (c *Cache) Update(ctx context.Context, repos []Repository) error {
	groupedRepos := make(map[string][]Repository)
	for _, repo := range repos {
		provider := repo.Provider()

		_, ok := groupedRepos[provider]
		if !ok {
			groupedRepos[provider] = make([]Repository, 0)
		}
		groupedRepos[provider] = append(groupedRepos[provider], repo)
	}

	output, err := json.MarshalIndent(groupedRepos, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	if err := os.MkdirAll(c.location, 0o755); err != nil {
		return fmt.Errorf("failed to create cache location: %w", err)
	}

	file, err := os.Create(c.path())
	if err != nil {
		return fmt.Errorf("failed to create cache file: %w", err)
	}

	_, err = file.Write(output)
	if err != nil {
		return fmt.Errorf("failed to write to cache: %w", err)
	}

	return nil
}

func (c *Cache) Clear(ctx context.Context) error {
	if err := os.Remove(c.path()); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}

		return err
	}

	return nil
}

func (c *Cache) path() string {
	return path.Join(c.location, c.fileName)
}

type GitHubProvider struct {
}

func NewGitHubProvider() *GitHubProvider {
	return &GitHubProvider{}
}

func getOrgRepos(ctx context.Context, client *github.Client, page int) ([]*github.Repository, error) {
	log.Printf("sending request for page: %d", page)
	repos, resp, err := client.Repositories.ListByAuthenticatedUser(ctx, &github.RepositoryListByAuthenticatedUserOptions{
		Visibility:  "all",
		Sort:        "updated",
		Affiliation: "organization_member",
		ListOptions: github.ListOptions{Page: page, PerPage: 100},
	})
	if err != nil {
		return nil, err
	}

	if resp.NextPage == 0 {
		return repos, nil
	}

	moreRepos, err := getOrgRepos(ctx, client, resp.NextPage)
	if err != nil {
		return nil, err
	}

	return append(repos, moreRepos...), nil
}

func getUserRepos(ctx context.Context, client *github.Client, page int) ([]*github.Repository, error) {
	log.Printf("sending request for page: %d", page)
	repos, resp, err := client.Repositories.ListByAuthenticatedUser(ctx, &github.RepositoryListByAuthenticatedUserOptions{
		Visibility:  "all",
		Sort:        "updated",
		Affiliation: "owner",
		ListOptions: github.ListOptions{Page: page, PerPage: 100},
	})
	if err != nil {
		return nil, err
	}

	if resp.NextPage == 0 {
		return repos, nil
	}

	moreRepos, err := getUserRepos(ctx, client, resp.NextPage)
	if err != nil {
		return nil, err
	}

	return append(repos, moreRepos...), nil
}

func (g *GitHubProvider) Get(ctx context.Context) ([]*GitHubRepository, error) {

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: os.Getenv("GITHUB_ACCESS_TOKEN")})
	httpClient := oauth2.NewClient(ctx, ts)

	client := github.NewClient(httpClient)

	repos, err := getOrgRepos(ctx, client, 0)
	if err != nil {
		return nil, err
	}

	gitHubRepos := make([]*GitHubRepository, 0, len(repos))
	for _, repo := range repos {
		gitHubRepos = append(gitHubRepos, NewGitHubRepository(repo))
	}

	repos, err = getUserRepos(ctx, client, 0)
	if err != nil {
		return nil, err
	}
	for _, repo := range repos {
		gitHubRepos = append(gitHubRepos, NewGitHubRepository(repo))
	}

	return gitHubRepos, nil
}

func main() {
	cmd := cobra.Command{
		Use: "fuzzy-clone",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			cache := NewCache()

			repos, exists, err := cache.Get(ctx)
			if err != nil {
				return fmt.Errorf("cache corrupted: %w", err)
			}

			if !exists {
				// 1. Gather
				gitHubRepos, err := NewGitHubProvider().Get(ctx)
				if err != nil {
					return fmt.Errorf("failed to get repos for github user: %w", err)
				}

				repos = toRepos(gitHubRepos)
			}

			// 2. Choose
			idx, err := fuzzyfinder.Find(
				repos,
				func(i int) string {
					return repos[i].ToString()
				},
			)
			if err != nil {
				if errors.Is(err, fuzzyfinder.ErrAbort) {
					os.Exit(1)
				}

				return err
			}

			repo := repos[idx]

			// 3. Clone
			destDir, err := repo.GetOrClone(ctx, "tmp")
			if err != nil {
				return fmt.Errorf("failed to get repository: %w", err)
			}

			// 4. Print location
			if _, err := fmt.Println(destDir); err != nil {
				return fmt.Errorf("failed to print destination directory: %w", err)
			}

			return nil
		},
	}

	cmd.AddCommand(cacheCommand())

	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func cacheCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "cache",
	}

	cmd.AddCommand(cacheUpdateCommand())
	cmd.AddCommand(cacheClearCommand())

	return cmd
}

func cacheUpdateCommand() *cobra.Command {
	return &cobra.Command{
		Use: "update",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			repos, err := NewGitHubProvider().Get(ctx)
			if err != nil {
				return fmt.Errorf("failed to get github user: %w", err)
			}

			cache := NewCache()

			if err := cache.Update(ctx, toRepos(repos)); err != nil {
				return fmt.Errorf("failed to update cache: %w", err)
			}

			return nil
		},
	}
}

func cacheClearCommand() *cobra.Command {
	return &cobra.Command{
		Use: "clear",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			cache := NewCache()

			if err := cache.Clear(ctx); err != nil {
				return fmt.Errorf("failed to clear cache: %w", err)
			}

			return nil
		},
	}
}
