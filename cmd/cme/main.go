package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/onceprgm/cme/internal/account"
	"github.com/onceprgm/cme/internal/clog"
	"github.com/onceprgm/cme/internal/installer"
	"github.com/onceprgm/cme/internal/launch"
	"github.com/onceprgm/cme/internal/manifest"
	"github.com/onceprgm/cme/internal/preflight"
	"github.com/onceprgm/cme/internal/store"
	"github.com/onceprgm/cme/internal/ui"
)

const usage = `cme - minimal Minecraft launcher for Linux

Usage:
  cme version list [--release|--snapshot|--old-beta|--old-alpha]
  cme install <version>
  cme install fabric|quilt <version> [loader]
  cme launch <version> --username <name> [--ram <GB>]
  cme launch fabric|quilt <version> [loader] --username <name> [--ram <GB>]
  cme verify <version>
  cme help

Global flags:
  -v, --verbose, --debug    mirror the detailed launcher log to stderr

The full launcher log is always written to $XDG_STATE_HOME/cme/cme.log.
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
  cme install fabric|quilt <version> [loader]

Downloads the client JAR, libraries, native libraries and assets for the given
version, all verified by SHA-1. Already-present files are skipped.

With 'fabric' or 'quilt', the vanilla base is installed first, then the loader
profile is fetched and its libraries downloaded. Without a loader version, the
latest stable one is used. Examples:

  cme install 1.20.1
  cme install fabric 1.21.4
  cme install quilt 1.21.4
  cme install fabric 1.21.4 0.16.9
`

const launchUsage = `cme launch - run an installed version in offline mode

Usage:
  cme launch <version> --username <name> [--ram <GB>]
  cme launch fabric|quilt <version> [loader] --username <name> [--ram <GB>]

Flags:
  --username <name>   player name (required; offline mode)
  --ram <GB>          memory in gigabytes, sets -Xmx and -Xms (optional)
  --jvm-arg <arg>     extra JVM argument, repeatable (advanced)

The version must be installed first with 'cme install'. With 'fabric' or 'quilt'
and no loader version, the installed loader is used (cme asks if several are
present). Examples:

  cme launch 1.20.1 --username Steve --ram 4
  cme launch fabric 1.21.4 --username Steve --ram 4
  cme launch quilt 1.21.4 --username Steve --ram 4
`

const verifyUsage = `cme verify - check an installed version and repair broken files

Usage:
  cme verify <version>
  cme verify fabric|quilt <version> [loader]

Re-checks the client JAR, libraries and assets of an installed version against
their SHA-1 and size, then re-downloads only the corrupt or missing files. The
check runs from local metadata (works offline); only repairs need the network.
Shared assets are checked by this version's index alone, so other versions are
never touched. Examples:

  cme verify 1.20.1
  cme verify fabric 1.21.4
`

func main() {
	os.Exit(mainCode())
}

func mainCode() int {
	verbose, args := splitVerbose(os.Args[1:])

	closeLog, err := clog.Setup(verbose)
	if err != nil {
		fmt.Fprintln(os.Stderr, "cme:", err)
		return 1
	}
	defer closeLog()

	if err := run(args); err != nil {
		clog.Error("command failed", "err", err.Error())
		fmt.Fprintln(os.Stderr, "cme:", err)
		return 1
	}
	return 0
}

func splitVerbose(args []string) (bool, []string) {
	verbose := os.Getenv("CME_DEBUG") != ""
	rest := make([]string, 0, len(args))
	for _, a := range args {
		switch a {
		case "-v", "--verbose", "--debug":
			verbose = true
		default:
			rest = append(rest, a)
		}
	}
	return verbose, rest
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
	case "verify":
		return cmdVerify(args[1:])
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

	if args[0] != "list" {
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

	if len(args) >= 1 && (args[0] == "fabric" || args[0] == "quilt") {
		return cmdInstallModded(args[0], args[1:])
	}

	if len(args) != 1 {
		return fmt.Errorf("usage: cme install <version> | cme install fabric|quilt <version> [loader]")
	}
	id := args[0]

	if err := preflight.RequireOnline(); err != nil {
		return err
	}

	m, err := manifest.FetchFresh()
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

func cmdInstallModded(kind string, args []string) error {
	if len(args) < 1 || len(args) > 2 {
		return fmt.Errorf("usage: cme install %s <version> [loader]", kind)
	}
	game := args[0]
	loader := ""
	if len(args) == 2 {
		loader = args[1]
	}

	install := installer.InstallFabric
	if kind == "quilt" {
		install = installer.InstallQuilt
	}

	if err := preflight.RequireOnline(); err != nil {
		return err
	}

	m, err := manifest.FetchFresh()
	if err != nil {
		return err
	}

	v := m.Find(game)
	if v == nil {
		return fmt.Errorf("version %q not found, try: cme version list", game)
	}

	ui.Info("installing %s for %s", kind, game)
	meta, err := install(v, loader, func(stage string, done, total int) {
		ui.Progress(stage, done, total)
	})
	if err != nil {
		return err
	}
	ui.Success("installed %s (requires java %d)", meta.ID, meta.JavaVersion.MajorVersion)
	ui.Info("launch it with: cme launch %s %s --username <name>", kind, game)
	return nil
}

func resolveLaunchTarget(args []string) (string, []string, error) {
	kind := args[0]
	if kind != "fabric" && kind != "quilt" {
		return args[0], args[1:], nil
	}
	if len(args) < 2 {
		return "", nil, fmt.Errorf("usage: cme launch %s <version> [loader] --username <name>", kind)
	}
	game := args[1]
	rest := args[2:]
	loader := ""
	if len(rest) > 0 && !strings.HasPrefix(rest[0], "-") {
		loader = rest[0]
		rest = rest[1:]
	}
	id, err := resolveModdedID(kind, game, loader)
	return id, rest, err
}

func resolveModdedID(kind, game, loader string) (string, error) {
	prefix := kind + "-loader-"
	if loader != "" {
		return prefix + loader + "-" + game, nil
	}

	suffix := "-" + game
	var matches []string
	entries, _ := os.ReadDir(store.VersionsDir())
	for _, e := range entries {
		n := e.Name()
		if e.IsDir() && strings.HasPrefix(n, prefix) && strings.HasSuffix(n, suffix) {
			matches = append(matches, n)
		}
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("no %s install for %s; run: cme install %s %s", kind, game, kind, game)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("multiple %s loaders for %s: %s; specify one, e.g. cme launch %s %s <loader>",
			kind, game, strings.Join(matches, ", "), kind, game)
	}
}

func cmdVerify(args []string) error {
	if wantsHelp(args) {
		fmt.Print(verifyUsage)
		return nil
	}
	if len(args) < 1 {
		return fmt.Errorf("usage: cme verify <version> | cme verify fabric|quilt <version> [loader]")
	}

	id, _, err := resolveLaunchTarget(args)
	if err != nil {
		return err
	}

	ui.Info("verifying %s", id)
	report, err := installer.Verify(id, func(stage string, done, total int) {
		ui.Progress(stage, done, total)
	})
	if err != nil {
		return err
	}

	ui.Success("verified %s: %d ok, %d repaired", id, report.OK, report.Repaired)
	if report.LoaderPresent > 0 || report.LoaderMissing > 0 {
		ui.Info("loader libraries: %d present, %d refetched (existence-checked)", report.LoaderPresent, report.LoaderMissing)
	}
	return nil
}

func cmdLaunch(args []string) error {
	if wantsHelp(args) {
		fmt.Print(launchUsage)
		return nil
	}

	if len(args) < 1 {
		return fmt.Errorf("usage: cme launch <version> --username <name> [--ram <GB>] [--jvm-arg <arg>]")
	}

	id, rest, err := resolveLaunchTarget(args)
	if err != nil {
		return err
	}

	username := ""
	ram := ""
	var extraJVM []string

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
		case "--jvm-arg":
			if i+1 >= len(rest) {
				return fmt.Errorf("--jvm-arg needs a value")
			}
			extraJVM = append(extraJVM, rest[i+1])
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
	} else {
		ui.Warn("no --ram set; using JVM defaults, which may be too low for modern versions")
	}
	jvmArgs = append(jvmArgs, extraJVM...)

	return launch.Launch(launch.Options{
		VersionID: id,
		Account:   account.Offline(username),
		JVMArgs:   jvmArgs,
	})
}
