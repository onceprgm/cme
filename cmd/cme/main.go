package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/onceprgm/cme/internal/download"
	"github.com/onceprgm/cme/internal/manifest"
	"github.com/onceprgm/cme/internal/store"
)

const usage = `cme - minimal Minecraft launcher for Linux

Usage:
  cme version list [--release|--snapshot|--old-beta|--old-alpha]
  cme install <version>
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

	fmt.Fprintf(os.Stderr, "resolving %s...\n", id)
	meta, raw, err := manifest.FetchVersionMeta(v)
	if err != nil {
		return err
	}

	dir := store.VersionDir(id)
	if err := store.Ensure(dir); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, id+".json"), raw, 0o644); err != nil {
		return err
	}

	jar := filepath.Join(dir, id+".jar")
	fmt.Fprintf(os.Stderr, "downloading client.jar (%.1f MB)...\n",
		float64(meta.Downloads.Client.Size)/1024/1024)
	if err := download.File(meta.Downloads.Client.URL, jar, meta.Downloads.Client.SHA1); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "installed %s (java %d required)\n", id, meta.JavaVersion.MajorVersion)
	// TODO: libraries, asset index, natives
	return nil
}
