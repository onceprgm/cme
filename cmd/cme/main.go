package main

import (
	"fmt"
	"os"

	"github.com/onceprgm/cme/internal/manifest"
)

const usage = `cme - minimal Minecraft launcher for Linux

Usage:
  cme version list [--release|--snapshot|--old-beta|--old-alpha]
  cme help

More commands (install, launch, profile) are on the way.
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
