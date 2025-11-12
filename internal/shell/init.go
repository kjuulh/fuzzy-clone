package shell

import (
	"context"
	"fmt"
	configfile "fuzzy-clone/internal/config"
	"fuzzy-clone/shell/config"
	"fuzzy-clone/shell/fish"
	"fuzzy-clone/shell/zsh"
	"os"

	"github.com/urfave/cli/v3"
)

func InitCmd() *cli.Command {
	return &cli.Command{
		Name:  "init",
		Usage: "Initialize shell integration",
		Commands: []*cli.Command{
			zshCmd(),
			fishCmd(),
			configCmd(),
		},
	}
}

func zshCmd() *cli.Command {
	return &cli.Command{
		Name:  "zsh",
		Usage: "Output zsh initialization script",
		Action: func(ctx context.Context, c *cli.Command) error {
			zshScript := zsh.Get()
			fmt.Println(zshScript)
			return nil
		},
	}
}

func fishCmd() *cli.Command {
	return &cli.Command{
		Name:  "fish",
		Usage: "Output fish initialization script",
		Action: func(ctx context.Context, c *cli.Command) error {
			fishScript := fish.MustGet()
			fmt.Println(fishScript)
			return nil
		},
	}
}

func configCmd() *cli.Command {
	var (
		write bool
	)

	return &cli.Command{
		Name:  "config",
		Usage: "Outputs a default template for the config",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "write",
				Usage:       "Should we write the default config to ~/.config/fz/config.toml",
				Destination: &write,
			},
		},
		Action: func(ctx context.Context, c *cli.Command) error {
			configContent := config.Get()

			if write {
				if err := os.MkdirAll(configfile.Parent(), 0o755); err != nil {
					return fmt.Errorf("create parent dir for config file: %w", err)
				}

				file, err := os.Create(configfile.Path())
				if err != nil {
					return fmt.Errorf("create config file: %w", err)
				}

				_, err = file.WriteString(configContent)
				if err != nil {
					return fmt.Errorf("write config file: %w", err)
				}

				fmt.Fprintf(c.ErrWriter, "created config file at: %s\n", configfile.Path())

				return nil
			}

			fmt.Fprintln(c.ErrWriter, configContent)
			fmt.Fprintln(c.ErrWriter, "\nuse --write to persist the example, you can always change it")

			return nil
		},
	}
}
