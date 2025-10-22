package web

import "sync"

var (
	verMu     sync.RWMutex
	ver       = "dev"
	verCommit = "unknown"
	verBuild  = "unknown"
)

// SetVersionInfo sets the version information to be exposed by the web API
func SetVersionInfo(versionStr, commit, buildTime string) {
	verMu.Lock()
	defer verMu.Unlock()
	ver = versionStr
	verCommit = commit
	verBuild = buildTime
}

// GetVersionInfo returns the currently set version info
func GetVersionInfo() (string, string, string) {
	verMu.RLock()
	defer verMu.RUnlock()
	return ver, verCommit, verBuild
}
