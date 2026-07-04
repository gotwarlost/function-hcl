package main

import (
	"encoding/json"
	"fmt"
	"runtime"
	"runtime/debug"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var Version = ""

type buildInfo struct {
	Version   string `json:"version,omitempty"`
	GoVersion string `json:"go,omitempty"`
	GoOS      string `json:"os,omitempty"`
	GoArch    string `json:"arch,omitempty"`
}

func getBuildInfo() buildInfo {
	output := buildInfo{
		Version:   "dev",
		GoVersion: "unknown",
		GoOS:      runtime.GOOS,
		GoArch:    runtime.GOARCH,
	}

	info, ok := debug.ReadBuildInfo()
	if ok {
		if Version == "" {
			if info.Main.Version != "" && info.Main.Version != "(devel)" {
				output.Version = info.Main.Version
			}
		} else {
			output.Version = Version
		}
		output.GoVersion = info.GoVersion
	}
	return output
}

func showVersion(jsonOutput bool) error {
	output := getBuildInfo()
	if jsonOutput {
		out, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return errors.Wrap(err, "marshal version output")
		}
		fmt.Println(string(out))
		return nil
	}

	fmt.Printf("%s\nplatform: %s/%s\ngo: %s\n",
		output.Version, output.GoOS, output.GoArch, output.GoVersion)
	return nil
}

// AddVersionCommand adds a sub-command to display version information.
func AddVersionCommand(root *cobra.Command) {
	var jsonOutput bool
	c := &cobra.Command{
		Use:   "version",
		Short: `display version and build information`,
	}
	root.AddCommand(c)
	f := c.Flags()
	f.BoolVar(&jsonOutput, "json", false, "output the version information as a JSON object")
	c.RunE = func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		return showVersion(jsonOutput)
	}
}
