package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"fuzzy-clone/internal/shell"

	"github.com/BurntSushi/toml"
	"github.com/adrg/xdg"
	"github.com/google/go-github/v60/github"
	"github.com/ktr0731/go-fuzzyfinder"
	altsrc "github.com/urfave/cli-altsrc/v3"
	"github.com/urfave/cli/v3"
	"golang.org/x/oauth2"
)

type Repository interface {
	Provider() string
	GetOrClone(ctx context.Context, config *Config) (string, error)
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

func (g *GitHubRepository) GetOrClone(ctx context.Context, config *Config) (string, error) {
	destDir, err := g.getFilePath(config)
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(destDir); errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(destDir, 0o755); err != nil {
			return "", fmt.Errorf("prepare git dir: %w", err)
		}
	}

	entries, err := os.ReadDir(destDir)
	if err != nil {
		return "", fmt.Errorf("read destination dir entries in %s: %w", destDir, err)
	}

	if len(entries) == 0 {
		tryClone := func(url string) error {
			process := exec.Command("git", "clone", url, ".")
			process.Dir = destDir

			if err := process.Start(); err != nil {
				return err
			}

			fmt.Fprintln(os.Stderr, "Downloading...")

			if err := process.Wait(); err != nil {
				return err
			}

			return nil
		}

		if g.SshUrl != nil {
			if err := tryClone(*g.SshUrl); err == nil {
				return destDir, nil
			}

			fmt.Fprintf(os.Stderr, "failed to clone with ssh, falling back to http: %s\n", err)
		}

		if g.HttpsUrl != nil {
			if err := tryClone(*g.HttpsUrl); err == nil {
				return destDir, nil
			}

			return "", fmt.Errorf("clone with http: %w", err)
		}

		return "", fmt.Errorf("clone repository: %s", g.FullName)
	}

	return destDir, nil
}

func (g *GitHubRepository) getFilePath(config *Config) (string, error) {
	var (
		destDir string
	)

	// $pwd/fuzzy-clone
	if config.UseCwd {
		cwdPath, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("get working directory: %w", err)
		}

		_, repoName := path.Split(g.FullName)
		destDir = path.Join(cwdPath, repoName)

		return destDir, nil
	}

	// ~/git/fuzzy-clone
	if config.FlattenDestination {
		_, repoName := path.Split(g.FullName)
		destDir = path.Join(getHomeOrDefault(config), repoName)
		return destDir, nil
	}

	// Full path ~/git/github.com/kjuulh/fuzzy-clone
	return path.Join(getHomeOrDefault(config), g.Provider(), g.FullName), nil
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

		return nil, false, fmt.Errorf("read cache file: %w", err)
	}

	var groupedRepos cachedRepositories
	if err := json.Unmarshal(contents, &groupedRepos); err != nil {
		return nil, false, fmt.Errorf("unmarshal cached repos: %s, %w", c.path(), err)
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
		return fmt.Errorf("marshal cache: %w", err)
	}

	if err := os.MkdirAll(c.location, 0o755); err != nil {
		return fmt.Errorf("create cache location: %w", err)
	}

	file, err := os.Create(c.path())
	if err != nil {
		return fmt.Errorf("create cache file: %w", err)
	}

	_, err = file.Write(output)
	if err != nil {
		return fmt.Errorf("write to cache: %w", err)
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

func (c *Cache) timestampPath() string {
	return path.Join(c.location, "last_update")
}

func (c *Cache) needsUpdate() (bool, error) {
	timestampBytes, err := os.ReadFile(c.timestampPath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// No timestamp file means we need to update
			return true, nil
		}
		return false, fmt.Errorf("read timestamp file: %w", err)
	}

	lastUpdate, err := time.Parse(time.RFC3339, strings.TrimSpace(string(timestampBytes)))
	if err != nil {
		slog.Debug("Parsing timestamp to RFC3339 failed", "timestamp", timestampBytes, "error", err.Error())
		// If we can't parse the timestamp, assume we need to update
		return true, nil
	}

	// Check if 24 hours have passed
	return time.Since(lastUpdate) >= 24*time.Hour, nil
}

func (c *Cache) updateTimestamp() error {
	if err := os.MkdirAll(c.location, 0o755); err != nil {
		return fmt.Errorf("create cache folder: %w", err)
	}

	timestamp := time.Now().Format(time.RFC3339)
	if err := os.WriteFile(c.timestampPath(), []byte(timestamp), 0o644); err != nil {
		return fmt.Errorf("write timestamp file: %w", err)
	}

	return nil
}

type GitHubProvider struct{}

func NewGitHubProvider() *GitHubProvider {
	return &GitHubProvider{}
}

func getOrgRepos(ctx context.Context, client *github.Client, page int) ([]*github.Repository, error) {
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

var (
	ErrNoTokenFound = errors.New("no Github token found")
	ErrUnknown      = errors.New("unknown error")
)

type Config struct {
	GitHubToken        string
	GitHubAccessToken  string
	Root               string
	CacheCooldown      string
	UseCwd             bool
	FlattenDestination bool
}

func getGitHubToken(cfg *Config) (string, error) {
	if cfg.GitHubToken != "" {
		return cfg.GitHubToken, nil
	}

	if cfg.GitHubAccessToken != "" {
		return cfg.GitHubAccessToken, nil
	}

	output, err := exec.Command("gh", "auth", "token").Output()
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrUnknown, err)
	} else {
		if len(output) != 0 {
			return strings.Replace(string(output), "\n", "", 1), nil // exec.Command appends a "\n"... Remove this
		}
	}

	return "", ErrNoTokenFound
}

func getHomeOrDefault(cfg *Config) string {
	if cfg.Root != "" {
		return cfg.Root
	}

	return os.ExpandEnv("$HOME/git")
}

func (g *GitHubProvider) Get(ctx context.Context, cfg *Config) ([]*GitHubRepository, error) {
	token, err := getGitHubToken(cfg)
	if token == "" {
		fmt.Fprintf(os.Stderr, "auth error: %s\n", err)
		return nil, fmt.Errorf("a token is required for github, follow setup in readme, and remember that the token should have at least repo:read, or consider installing the github-cli (gh) utility")
	}

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	httpClient := oauth2.NewClient(ctx, ts)

	client := github.NewClient(httpClient)

	fmt.Fprintln(os.Stderr, "fetching github repos, this may take a bit...")
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

var (
	configFile = altsrc.StringSourcer(path.Join(os.ExpandEnv("$HOME/.config"), "fz", "config.toml"))
)

func tomlSource(key string) *altsrc.ValueSource {
	return altsrc.NewValueSource(toml.Unmarshal, "toml", key, configFile)
}

func main() {

	var cfg Config

	app := &cli.Command{
		Name:  "fuzzy-clone",
		Usage: "Fuzzy find and clone repositories",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "use-cwd",
				Aliases:     []string{"c"},
				Usage:       "when set, clone repo into CWD",
				Sources:     cli.NewValueSourceChain(cli.EnvVar("FUZZY_CLONE_USE_CWD"), tomlSource("use_cwd")),
				Destination: &cfg.UseCwd,
			},
			&cli.BoolFlag{
				Name:        "flatten-destination",
				Usage:       "when set, the destination path will no longer be namespaces, and instead just put in the root folder",
				Sources:     cli.NewValueSourceChain(cli.EnvVar("FUZZY_CLONE_FLATTEN_DESTINATION"), tomlSource("flatten_destination")),
				Destination: &cfg.FlattenDestination,
			},
			&cli.StringFlag{
				Name:        "github-token",
				Usage:       "GitHub personal access token",
				Sources:     cli.NewValueSourceChain(cli.EnvVar("FUZZY_CLONE_GITHUB_TOKEN"), tomlSource("github.token")),
				Destination: &cfg.GitHubToken,
			},
			&cli.StringFlag{
				Name:        "github-access-token",
				Usage:       "GitHub access token (alternative)",
				Sources:     cli.EnvVars("GITHUB_ACCESS_TOKEN"),
				Destination: &cfg.GitHubAccessToken,
			},
			&cli.StringFlag{
				Name:        "root",
				Usage:       "Root directory for cloning repositories",
				Sources:     cli.NewValueSourceChain(cli.EnvVar("FUZZY_CLONE_ROOT"), tomlSource("root")),
				Destination: &cfg.Root,
			},
			&cli.StringFlag{
				Name:        "cache-cooldown",
				Usage:       "Enable cache cooldown (true/false)",
				Sources:     cli.NewValueSourceChain(cli.EnvVar("FUZZY_CLONE_CACHE_COOLDOWN"), tomlSource("cache_cooldown")),
				Destination: &cfg.CacheCooldown,
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			cache := NewCache()

			repos, exists, err := cache.Get(ctx)
			if err != nil {
				return fmt.Errorf("cache corrupted: %w", err)
			}

			if !exists {
				// 1. Gather
				gitHubRepos, err := NewGitHubProvider().Get(ctx, &cfg)
				if err != nil {
					return fmt.Errorf("retrieve list of repos for github user: %w", err)
				}

				repos = toRepos(gitHubRepos)
			}

			providers := make(map[string]struct{})

			for _, repo := range repos {
				providers[repo.Provider()] = struct{}{}
			}

			// 2. Choose
			idx, err := fuzzyfinder.Find(
				repos,
				func(i int) string {
					if len(providers) == 1 {
						return strings.TrimPrefix(repos[i].ToString(), fmt.Sprintf("%s/", repos[i].Provider()))
					}

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
			destDir, err := repo.GetOrClone(ctx, &cfg)
			if err != nil {
				return fmt.Errorf("clone repository: %w", err)
			}

			// 4. Print location
			if _, err := fmt.Println(destDir); err != nil {
				return fmt.Errorf("print destination directory: %w", err)
			}

			return nil
		},
		Commands: []*cli.Command{
			cacheCommand(&cfg),
			shell.InitCmd(),
			&cli.Command{
				Name: "doctor",
				Action: func(ctx context.Context, c *cli.Command) error {
					fmt.Fprintf(c.Writer, "config_file: %s, exists: %s\n", configFile.SourceURI(), configFileExists())

					return nil
				},
			},
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func configFileExists() string {
	_, err := os.Stat(configFile.SourceURI())
	if errors.Is(err, os.ErrNotExist) {
		return "nope"
	}

	if err != nil {
		return err.Error()
	}

	return "yep"
}

func cacheCommand(cfg *Config) *cli.Command {
	return &cli.Command{
		Name:  "cache",
		Usage: "Manage repository cache",
		Commands: []*cli.Command{
			cacheUpdateCommand(cfg),
			cacheClearCommand(),
		},
	}
}

func cacheUpdateCommand(cfg *Config) *cli.Command {
	var force bool

	return &cli.Command{
		Name:  "update",
		Usage: "Update the repository cache",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "force",
				Aliases:     []string{"f"},
				Usage:       "force cache update even if updated within the last 24 hours",
				Destination: &force,
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			cache := NewCache()

			// Check if cache needs updating (unless --force is set)
			// Only apply cooldown if FUZZY_CLONE_CACHE_COOLDOWN is set to "true"
			if !force && cfg.CacheCooldown == "true" {
				needsUpdate, err := cache.needsUpdate()
				if err != nil {
					return fmt.Errorf("check cache needs update: %w", err)
				}

				if !needsUpdate {
					// Cache was updated within the last 24 hours, skip update
					return nil
				}
			}

			repos, err := NewGitHubProvider().Get(ctx, cfg)
			if err != nil {
				return fmt.Errorf("get github repo list for user: %w", err)
			}

			if err := cache.Update(ctx, toRepos(repos)); err != nil {
				return fmt.Errorf("update cache: %w", err)
			}

			// Update the timestamp after successful cache update
			if err := cache.updateTimestamp(); err != nil {
				return fmt.Errorf("update cache timestamp: %w", err)
			}

			return nil
		},
	}
}

func cacheClearCommand() *cli.Command {
	return &cli.Command{
		Name:  "clear",
		Usage: "Clear the repository cache",
		Action: func(ctx context.Context, c *cli.Command) error {
			cache := NewCache()

			if err := cache.Clear(ctx); err != nil {
				return fmt.Errorf("clearing cache: %w", err)
			}

			return nil
		},
	}
}
