package shell

import (
	"fmt"
	"fuzzy-clone/shell/zsh"

	"github.com/spf13/cobra"
)

func InitCmd() *cobra.Command {
	cmd := cobra.Command{
		Use: "init",
	}

	cmd.AddCommand(
		zshCmd(),
	)

	return &cmd
}

func zshCmd() *cobra.Command {
	return &cobra.Command{
		Use: "zsh",
		Run: func(cmd *cobra.Command, args []string) {
			zshScript := zsh.Get()

			fmt.Println(zshScript)
		},
	}
}
