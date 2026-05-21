package version

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

func String() string {
	if Commit == "none" {
		return Version
	}
	return Version + " (" + Commit + ")"
}
