package config

import (
	"os"
	"path"

	altsrc "github.com/urfave/cli-altsrc/v3"
)

var (
	ConfigFile = altsrc.StringSourcer(
		path.Join(
			os.ExpandEnv("$HOME/.config"),
			"fz",
			"config.toml"),
	)
)

func Path() string {
	return ConfigFile.SourceURI()
}

func Parent() string {
	return path.Dir(Path())
}
