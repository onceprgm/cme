package main

import (
	"fmt"
	"os"
	"strconv"
	"text/tabwriter"

	"github.com/onceprgm/cme/internal/java"
	"github.com/onceprgm/cme/internal/preflight"
	"github.com/onceprgm/cme/internal/ui"
)

const javaUsage = `cme java - manage Java runtimes

Usage:
  cme java install <major>
  cme java list

'install' downloads an Eclipse Temurin JRE for the given major version from
Adoptium, verifies its SHA-256, and unpacks it into the cme data directory.
cme then finds it automatically when a version needs that Java. Examples:

  cme java install 21
  cme java list
`

func cmdJava(args []string) error {
	if len(args) == 0 || wantsHelp(args) {
		fmt.Print(javaUsage)
		return nil
	}
	switch args[0] {
	case "install":
		return javaInstall(args[1:])
	case "list", "ls":
		return javaList(args[1:])
	default:
		return fmt.Errorf("unknown java command %q (try: install, list)", args[0])
	}
}

func javaInstall(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: cme java install <major>")
	}
	major, err := strconv.Atoi(args[0])
	if err != nil || major <= 0 {
		return fmt.Errorf("major must be a positive integer, got %q", args[0])
	}

	if err := preflight.RequireOnline(); err != nil {
		return err
	}

	ui.Info("installing java %d (Temurin JRE)", major)
	lastMB := -1
	bin, err := java.Install(major, func(done, total int64) {
		mb := int(done >> 20)
		if mb != lastMB {
			lastMB = mb
			ui.Progress("java", mb, int(total>>20))
		}
	})
	if err != nil {
		return err
	}
	ui.Success("installed java %d at %s", major, bin)
	return nil
}

func javaList(args []string) error {
	list, err := java.List()
	if err != nil {
		return err
	}
	if hasJSON(args) {
		return printJSON(list)
	}
	if len(list) == 0 {
		ui.Info("no managed java installs; add one: cme java install 21")
		return nil
	}
	tw := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	for _, j := range list {
		fmt.Fprintf(tw, "java %d\t%s\n", j.Major, j.Path)
	}
	return tw.Flush()
}
