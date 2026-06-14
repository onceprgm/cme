package main

import (
	"fmt"
	"os"
	"strconv"
	"text/tabwriter"

	"github.com/onceprgm/cme/internal/account"
	"github.com/onceprgm/cme/internal/installer"
	"github.com/onceprgm/cme/internal/launch"
	"github.com/onceprgm/cme/internal/manifest"
	"github.com/onceprgm/cme/internal/preflight"
	"github.com/onceprgm/cme/internal/ui"
)

const usage = `cme - minimal Minecraft launcher for Linux

Usage:
  cme version list [--release|--snapshot|--old-beta|--old-alpha]
  cme install <version>
  cme launch <version> --username <name> [--ram <GB>]
  cme help
`

const versionUsage = `cme version list - list available Minecraft versions

Usage:
  cme version list [filter]

Filters:
  --release      stable releases only
  --snapshot     development snapshots only
  --old-beta     old beta versions (2010-2011)
  --old-alpha    old alpha versions (2010)

With no filter, every version is listed. A * marks the latest release/snapshot.
`

const installUsage = `cme install - download a Minecraft version

Usage:
  cme install <version>

Downloads the client JAR, libraries, native libraries and assets for the given
version, all verified by SHA-1. Already-present files are skipped. Example:

  cme install 1.20.1
`

const launchUsage = `cme launch - run an installed version in offline mode

Usage:
  cme launch <version> --username <name> [--ram <GB>]

Flags:
  --username <name>   player name (required; offline mode)
  --ram <GB>          memory in gigabytes, sets -Xmx and -Xms (optional)

The version must be installed first with 'cme install'. Example:

  cme launch 1.20.1 --username Steve --ram 4
`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "cme:", err)
		os.Exit(1)
	}
}

func wantsHelp(args []string) bool {
	for _, a := range args {
		if a == "--help" || a == "-h" {
			return true
		}
	}
	return false
}

func run(args []string) error {
	if len(args) == 0 {
		fmt.Print(usage)
		return nil
	}

	switch args[0] {
	case "version":
		return cmdVersion(args[1:])
	case "install":
		return cmdInstall(args[1:])
	case "launch":
		return cmdLaunch(args[1:])
	case "help", "--help", "-h":
		fmt.Print(usage)
		return nil
	default:
		fmt.Print(usage)
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func cmdVersion(args []string) error {
	if len(args) == 0 || wantsHelp(args) {
		fmt.Print(versionUsage)
		return nil
	}

	if len(args) == 0 || args[0] != "list" {
		return fmt.Errorf("usage: cme version list [--release|--snapshot|--old-beta|--old-alpha]")
	}

	var filter manifest.VersionType
	if len(args) > 1 {
		switch args[1] {
		case "--release":
			filter = manifest.TypeRelease
		case "--snapshot":
			filter = manifest.TypeSnapshot
		case "--old-beta":
			filter = manifest.TypeOldBeta
		case "--old-alpha":
			filter = manifest.TypeOldAlpha
		default:
			return fmt.Errorf("unknown flag %q", args[1])
		}
	}

	if err := preflight.RequireOnline(); err != nil {
		return err
	}

	m, err := manifest.Fetch()
	if err != nil {
		return err
	}

	tw := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	for _, v := range m.Filter(filter) {
		marker := " "
		if v.ID == m.Latest.Release || v.ID == m.Latest.Snapshot {
			marker = "*"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n",
			marker, v.ID, v.Type, v.ReleaseTime.Format("2006-01-02"))
	}
	return tw.Flush()
}

func cmdInstall(args []string) error {
	if wantsHelp(args) {
		fmt.Print(installUsage)
		return nil
	}

	if len(args) != 1 {
		return fmt.Errorf("usage: cme install <version>")
	}
	id := args[0]

	if err := preflight.RequireOnline(); err != nil {
		return err
	}

	m, err := manifest.Fetch()
	if err != nil {
		return err
	}

	v := m.Find(id)
	if v == nil {
		return fmt.Errorf("version %q not found, try: cme version list", id)
	}

	ui.Info("installing %s", id)
	meta, err := installer.Install(v, func(stage string, done, total int) {
		ui.Progress(stage, done, total)
	})
	if err != nil {
		return err
	}
	ui.Success("installed %s (requires java %d)", id, meta.JavaVersion.MajorVersion)
	return nil
}

func cmdLaunch(args []string) error {
	if wantsHelp(args) {
		fmt.Print(launchUsage)
		return nil
	}

	if len(args) < 1 {
		return fmt.Errorf("usage: cme launch <version> --username <name> [--ram <GB>]")
	}
	id := args[0]
	username := ""
	ram := ""

	rest := args[1:]
	for i := 0; i < len(rest); i++ {
		switch rest[i] {
		case "--username":
			if i+1 >= len(rest) {
				return fmt.Errorf("--username needs a value")
			}
			username = rest[i+1]
			i++
		case "--ram":
			if i+1 >= len(rest) {
				return fmt.Errorf("--ram needs a value")
			}
			ram = rest[i+1]
			if n, err := strconv.Atoi(ram); err != nil || n <= 0 {
				return fmt.Errorf("--ram must be a positive integer (GB), got %q", ram)
			}
			i++
		default:
			return fmt.Errorf("unknown flag %q", rest[i])
		}
	}

	if username == "" {
		return fmt.Errorf("--username is required (offline mode)")
	}

	var jvmArgs []string
	if ram != "" {
		jvmArgs = append(jvmArgs, "-Xmx"+ram+"G", "-Xms"+ram+"G")
	}

	return launch.Launch(launch.Options{
		VersionID: id,
		Account:   account.Offline(username),
		JVMArgs:   jvmArgs,
	})
}
