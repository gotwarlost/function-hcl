package cmd

import (
	"encoding/json"
	"fmt"
	"runtime/debug"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type versionOutput struct {
	Version string `json:"version"`
	buildInfo
}

type buildInfo struct {
	GoVersion string `json:"go,omitempty"`
	GoOS      string `json:"os,omitempty"`
	GoArch    string `json:"arch,omitempty"`
	Compiler  string `json:"compiler,omitempty"`
}

func showVersion(jsonOutput bool) error {
	info, ok := debug.ReadBuildInfo()
	output := versionOutput{
		Version: Version,
		buildInfo: buildInfo{
			GoVersion: "unknown",
			GoOS:      "unknown",
			GoArch:    "unknown",
			Compiler:  "unknown",
		},
	}
	if ok {
		output.GoVersion = info.GoVersion
		for _, setting := range info.Settings {
			// Filter for VCS-related info
			switch setting.Key {
			case "GOOS":
				output.GoOS = setting.Value
			case "GOARCH":
				output.GoArch = setting.Value
			case "compiler":
				output.Compiler = setting.Value
			}
		}
	}

	if jsonOutput {
		out, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			return errors.Wrap(err, "marshal version output")
		}
		fmt.Println(string(out))
		return nil
	}

	fmt.Printf("%s\nplatform: %s/%s\ngo: %s\ncompiler: %s\n",
		Version, output.GoOS, output.GoArch, output.GoVersion, output.Compiler)
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
