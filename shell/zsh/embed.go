package zsh

import "embed"

//go:embed fuzzy-clone.zsh
var ZshScript embed.FS

func Get() string {
	content, err := ZshScript.ReadFile("fuzzy-clone.zsh")
	if err != nil {
		panic(err)
	}

	return string(content)
}
