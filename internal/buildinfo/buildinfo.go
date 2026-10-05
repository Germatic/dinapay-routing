package buildinfo

import (
	"os"
	"runtime"
	"runtime/debug"
	"strings"
)

var (
	Version = "dev"
	Commit  = "unknown"
	BuiltAt = "unknown"
)

type Info struct {
	Service         string `json:"service"`
	Repository      string `json:"repository"`
	ContractVersion string `json:"contractVersion"`
	Version         string `json:"version"`
	Commit          string `json:"commit"`
	BuiltAt         string `json:"builtAt"`
	GoVersion       string `json:"goVersion"`
	Environment     string `json:"environment"`
}

func Current() Info {
	version, commit, builtAt := Version, Commit, BuiltAt
	if bi, ok := debug.ReadBuildInfo(); ok {
		if version == "dev" && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
			version = bi.Main.Version
		}
		if commit == "unknown" {
			for _, setting := range bi.Settings {
				if setting.Key == "vcs.revision" {
					commit = setting.Value
				}
				if setting.Key == "vcs.modified" && setting.Value == "true" {
					commit += "+dirty"
				}
			}
		}
	}
	environment := strings.TrimSpace(os.Getenv("ENVIRONMENT"))
	if environment == "" {
		environment = "unknown"
	}
	return Info{Service: "dinapay-routing", Repository: "github.com/Germatic/dinapay-routing", ContractVersion: "v2", Version: version, Commit: commit, BuiltAt: builtAt, GoVersion: runtime.Version(), Environment: environment}
}
