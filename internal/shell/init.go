package shell

import (
	"context"
	"fmt"
	"fuzzy-clone/shell/fish"
	"fuzzy-clone/shell/zsh"

	"github.com/urfave/cli/v3"
)

func InitCmd() *cli.Command {
	return &cli.Command{
		Name:  "init",
		Usage: "Initialize shell integration",
		Commands: []*cli.Command{
			zshCmd(),
			fishCmd(),
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
