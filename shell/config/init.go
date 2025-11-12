package config

const tomlExample = `
# use_cwd clones the choice into the current working directory
# use_cwd = false

# flatten_destination outputs the choice directly in the root, as opposed to the fully namespaced path
# ~/git/fuzzy-clone
# as opposed to
# ~/git/github.com/kjuulh/fuzzy-clone
# flatten_destination = false

# root places the choices in the root of choice. combine with above settings for more granular control
# root = "/home/kjuulh/git"

# [github]
# Optional token to use for authentication, else we fall back on the gh tool
# token = "my-token"

# [cache]
# By default fz hits the providers once pr call (in the background), adding cooldown awaits for a specific time before attempting to refresh in the background
# cooldown = false
`

func Get() string {
	return tomlExample
}
