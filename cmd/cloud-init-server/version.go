package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// GitCommit stores the latest Git commit hash.
// Set via -ldflags "-X main.GitCommit=$(git rev-parse HEAD)"
var GitCommit string

// BuildTime stores the build timestamp in UTC.
// Set via -ldflags "-X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
var BuildTime string

// Version indicates the version of the binary, such as a release number or semantic version.
// Set via -ldflags "-X main.Version=v1.0.0"
var Version string

// GitBranch holds the name of the Git branch from which the build was created.
// Set via -ldflags "-X main.GitBranch=$(git rev-parse --abbrev-ref HEAD)"
var GitBranch string

// GitTag represents the most recent Git tag at build time, if any.
// Set via -ldflags "-X main.GitTag=$(git describe --tags --abbrev=0)"
var GitTag string

// GitState indicates whether the working directory was "clean" or "dirty" (i.e., with uncommitted changes).
// Set via -ldflags "-X main.GitState=$(if git diff-index --quiet HEAD --; then echo 'clean'; else echo 'dirty'; fi)"
var GitState string

// BuildHost stores the hostname of the machine where the binary was built.
// Set via -ldflags "-X main.BuildHost=$(hostname)"
var BuildHost string

// GoVersion captures the Go version used to build the binary.
// Typically, this can be obtained automatically with runtime.Version(), but you can set it manually.
// Set via -ldflags "-X main.GoVersion=$(go version | awk '{print $3}')"
var GoVersion string

// BuildUser is the username of the person or system that initiated the build process.
// Set via -ldflags "-X main.BuildUser=$(whoami)"
var BuildUser string

var (
	startTime   time.Time
	runtimeHost string
	processName string
)

// PrintVersionInfo outputs all versioning information for troubleshooting or version checks.
func PrintVersionInfo() {
	fmt.Printf("Version: %s\n", Version)
	fmt.Printf("Git Commit: %s\n", GitCommit)
	fmt.Printf("Build Time: %s\n", BuildTime)
	fmt.Printf("Git Branch: %s\n", GitBranch)
	fmt.Printf("Git Tag: %s\n", GitTag)
	fmt.Printf("Git State: %s\n", GitState)
	fmt.Printf("Build Host: %s\n", BuildHost)
	fmt.Printf("Go Version: %s\n", GoVersion)
	fmt.Printf("Build User: %s\n", BuildUser)
}

func init() {
	startTime = time.Now()
	hostname, err := os.Hostname()
	if err == nil {
		runtimeHost = hostname
	} else {
		runtimeHost = "unknown"
	}
	processName = filepath.Base(os.Args[0])
}

func VersionInfo() string {
	return fmt.Sprintf("Version: %s, Git Commit: %s, Build Time: %s, Git Branch: %s, Git Tag: %s, Git State: %s, Build Host: %s, Go Version: %s, Build User: %s",
		Version, GitCommit, BuildTime, GitBranch, GitTag, GitState, BuildHost, GoVersion, BuildUser)
}

func VersionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	info := VersionInfo()
	uptime := time.Since(startTime).String()
	memstats := &runtime.MemStats{}
	runtime.ReadMemStats(memstats)

	response := struct {
		ProcessName    string `json:"process_name" yaml:"process_name"`
		BuildInfo      string `json:"build_info" yaml:"build_info"`
		Uptime         string `json:"uptime" yaml:"uptime"`
		RuntimeHost    string `json:"runtime_host" yaml:"runtime_host"`
		BytesAllocated uint64 `json:"bytes_allocated" yaml:"bytes_allocated"`
	}{
		ProcessName:    processName,
		BuildInfo:      info,
		Uptime:         uptime,
		RuntimeHost:    runtimeHost,
		BytesAllocated: memstats.HeapAlloc,
	}

	_ = json.NewEncoder(w).Encode(response) // Not checking error on Encode because we're wouldn't do anything about it anyway
}
