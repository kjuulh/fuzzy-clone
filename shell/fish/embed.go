package fish

import "embed"

//go:embed fuzzy-clone.fish
var FishScript embed.FS

func Get() string {
	content, err := FishScript.ReadFile("fuzzy-clone.fish")
	if err != nil {
		panic(err)
	}

	return string(content)
}
