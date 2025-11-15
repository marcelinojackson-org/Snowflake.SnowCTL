package build

var (
	// Version is injected via ldflags at release time.
	Version = "v0.0.1"
	// Commit represents the git SHA from which the binary was built.
	Commit = ""
	// Date holds the build timestamp.
	Date = ""
)
