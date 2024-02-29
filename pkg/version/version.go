package version

// variable is set to the correct version by the Dockerfile during build time.
var version = "dev"

func GetModuleVersion() string {
	return version
}
