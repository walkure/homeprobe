package revision

import (
	"flag"
	"fmt"
	"runtime"
)

// Build information. Populated at build-time.
var commit = "no commit"
var tag = "no tag"

// runtimeVersion is the version of the Go compiler used.
var runtimeVersion = fmt.Sprintf("%s %s/%s", runtime.Version(), runtime.GOOS, runtime.GOARCH)

// Usage returns a function that prints the usage and the build information.
func Usage(binname string) func() {
	return func() {
		fmt.Println("Built on", runtimeVersion, "at", commit, "/", tag)
		fmt.Printf("Usage: %s [options]\n", binname)
		flag.PrintDefaults()
	}
}
