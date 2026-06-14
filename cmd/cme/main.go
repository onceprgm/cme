package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/onceprgm/cme/internal/account"
	"github.com/onceprgm/cme/internal/installer"
	"github.com/onceprgm/cme/internal/launch"
	"github.com/onceprgm/cme/internal/manifest"
)

const usage = `cme - minimal Minecraft launcher for Linux

Usage:
  cme version list [--release|--snapshot|--old-beta|--old-alpha]
  cme install <version>
  cme launch <version> --username <name> [--ram <GB>]
  cme help
`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "cme:", err)
		os.Exit(1)
	}
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

	m, err := manifest.Fetch()
	if err != nil {
		return err
	}

	for _, v := range m.Filter(filter) {
		marker := " "
		if v.ID == m.Latest.Release || v.ID == m.Latest.Snapshot {
			marker = "*"
		}
		fmt.Printf("%s %-26s %-10s %s\n",
			marker, v.ID, v.Type, v.ReleaseTime.Format("2006-01-02"))
	}
	return nil
}

func cmdInstall(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: cme install <version>")
	}
	id := args[0]

	m, err := manifest.Fetch()
	if err != nil {
		return err
	}

	v := m.Find(id)
	if v == nil {
		return fmt.Errorf("version %q not found, try: cme version list", id)
	}

	lastStage := ""
	meta, err := installer.Install(v, func(stage string, done, total int) {
		if stage != lastStage {
			if lastStage != "" {
				fmt.Fprintln(os.Stderr)
			}
			lastStage = stage
		}
		fmt.Fprintf(os.Stderr, "\r%-10s %d/%d", stage, done, total)
	})
	if lastStage != "" {
		fmt.Fprintln(os.Stderr)
	}
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "installed %s (java %d required)\n", id, meta.JavaVersion.MajorVersion)
	return nil
}

func cmdLaunch(args []string) error {
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
