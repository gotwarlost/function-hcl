package main

import (
	"bytes"
	"fmt"
	"runtime"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

// Build information. Populated at build-time.
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

// info contains detailed information about the binary.
type buildInfo struct {
	version, commit, buildDate string
}

func (v buildInfo) String() string {
	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 2, 2, 3, ' ', 0)
	write := func(prompt, value string) {
		_, _ = fmt.Fprintln(w, prompt+"\t", value)
	}
	write("Version", v.version)
	write("Go Version", runtime.Version())
	write("Commit", strings.ReplaceAll(v.commit, "_", " "))
	write("Build Date", v.buildDate)
	write("OS/Arch", fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH))
	_ = w.Flush()
	return buf.String()
}

func versionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "print program version",
		Run: func(cmd *cobra.Command, args []string) {
			info := buildInfo{Version, Commit, BuildDate}
			fmt.Println(info)
		},
	}
}
