package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/onceprgm/cme/internal/account"
	"github.com/onceprgm/cme/internal/config"
	"github.com/onceprgm/cme/internal/installer"
	"github.com/onceprgm/cme/internal/launch"
	"github.com/onceprgm/cme/internal/manifest"
	"github.com/onceprgm/cme/internal/meta"
	"github.com/onceprgm/cme/internal/preflight"
	"github.com/onceprgm/cme/internal/profile"
	"github.com/onceprgm/cme/internal/store"
	"github.com/onceprgm/cme/internal/ui"
)

const profileUsage = `cme profile - manage named launch profiles

Usage:
  cme profile create <name> [fabric|quilt] <version> [loader] [flags]
  cme profile list
  cme profile show <name>
  cme profile delete <name>

Flags (create):
  --username <name>   player name (default: config username)
  --ram <GB>          memory in gigabytes (default: config ram)
  --jvm-arg <arg>     extra JVM argument, repeatable
  --dir <path>        game directory (default: instances/<name>)

A profile bundles a version, loader, account and JVM settings so you can launch
with a single 'cme run <name>'. Each profile gets its own game directory, so
saves, mods and options stay separate. With no version on a terminal, create
asks interactively. Examples:

  cme profile create modpack fabric 1.21.4 --ram 6 --username Steve
  cme profile create vanilla19 1.19.2
  cme run modpack
`

const runUsage = `cme run - launch a saved profile

Usage:
  cme run <profile> [--no-install]

Launches the profile created with 'cme profile create'. A missing username or
ram falls back to the global config (see 'cme config').

If the profile's version is not installed yet, cme offers to install it first
(on a terminal). Pass --no-install to fail instead, which suits scripts that
create, install and run as separate steps.
`

const configUsage = `cme config - global default settings

Usage:
  cme config set <key> <value>
  cme config get <key>
  cme config list

Keys:
  username   default player name used by profiles and 'cme launch'
  ram        default memory in gigabytes
  java       path to a Java binary to use instead of auto-detection

Examples:
  cme config set username Steve
  cme config set ram 4
`

func cmdProfile(args []string) error {
	if len(args) == 0 || wantsHelp(args) {
		fmt.Print(profileUsage)
		return nil
	}
	switch args[0] {
	case "create":
		return profileCreate(args[1:])
	case "list", "ls":
		return profileList()
	case "show":
		return profileShow(args[1:])
	case "delete", "rm":
		return profileDelete(args[1:])
	default:
		return fmt.Errorf("unknown profile command %q (try: create, list, show, delete)", args[0])
	}
}

func profileCreate(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cme profile create <name> [fabric|quilt] <version> [loader] [flags]")
	}
	name := args[0]
	if profile.Exists(name) {
		return fmt.Errorf("profile %q already exists", name)
	}

	loader, version, loaderVer, rest, err := parseTarget(args[1:])
	if err != nil {
		return err
	}
	username, ram, jvmArgs, dir, err := parseProfileFlags(rest)
	if err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	interactive := false
	if version == "" {
		if !ui.IsTerminal() {
			return fmt.Errorf("a version is required: cme profile create %s <version>", name)
		}
		loader, version, loaderVer, username, ram = profileWizard(cfg)
		if version == "" {
			return fmt.Errorf("aborted: no version given")
		}
		interactive = true
	}

	p := &profile.Profile{
		Name:          name,
		Loader:        loader,
		Version:       version,
		LoaderVersion: loaderVer,
		Username:      username,
		RAM:           ram,
		JVMArgs:       jvmArgs,
		GameDir:       dir,
	}

	if !interactive {
		if preflight.Online() {
			if err := validateTarget(p.Loader, p.Version); err != nil {
				return err
			}
		} else {
			ui.Warn("offline: could not verify %s exists; creating anyway", targetLabel(p))
		}
	}

	if err := profile.Save(p); err != nil {
		return err
	}
	ui.Success("created profile %q", name)
	printProfile(p, cfg)
	ui.Info("run it with: cme run %s", name)
	return nil
}

func profileWizard(cfg *config.Config) (loader, version, loaderVer, username string, ram int) {
	switch strings.ToLower(ui.Prompt("Loader: vanilla, fabric or quilt", "vanilla")) {
	case "fabric":
		loader = "fabric"
	case "quilt":
		loader = "quilt"
	}
	online := preflight.Online()
	for {
		version = ui.Prompt("Minecraft version", "")
		if version == "" {
			return
		}
		if !online {
			break
		}
		if err := validateTarget(loader, version); err != nil {
			ui.Warn("%s", err)
			continue
		}
		break
	}
	if loader != "" {
		loaderVer = ui.Prompt("Loader version (blank = latest stable)", "")
	}
	username = ui.Prompt("Username", cfg.Username)
	def := ""
	if cfg.RAM > 0 {
		def = strconv.Itoa(cfg.RAM)
	}
	if s := ui.Prompt("RAM in GB (blank = none)", def); s != "" {
		ram, _ = strconv.Atoi(s)
	}
	return
}

func profileList() error {
	profs, err := profile.List()
	if err != nil {
		return err
	}
	if len(profs) == 0 {
		ui.Info("no profiles yet; create one: cme profile create <name> <version>")
		return nil
	}
	tw := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, "NAME\tVERSION\tLOADER\tRAM\tUSERNAME")
	for _, p := range profs {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			p.Name, p.Version, dash(loaderLabel(p)), ramLabel(p.RAM), dash(p.Username))
	}
	return tw.Flush()
}

func profileShow(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: cme profile show <name>")
	}
	p, err := profile.Get(args[0])
	if err != nil {
		return err
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	printProfile(p, cfg)
	return nil
}

func profileDelete(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: cme profile delete <name>")
	}
	if err := profile.Delete(args[0]); err != nil {
		return err
	}
	ui.Success("deleted profile %q", args[0])
	return nil
}

func cmdRun(args []string) error {
	if wantsHelp(args) {
		fmt.Print(runUsage)
		return nil
	}

	name := ""
	noInstall := false
	for _, a := range args {
		switch a {
		case "--no-install":
			noInstall = true
		default:
			if name != "" {
				return fmt.Errorf("unexpected argument %q", a)
			}
			name = a
		}
	}
	if name == "" {
		return fmt.Errorf("usage: cme run <profile> [--no-install]")
	}

	p, err := profile.Get(name)
	if err != nil {
		return err
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	username := p.Username
	if username == "" {
		username = cfg.Username
	}
	if username == "" {
		return fmt.Errorf("no username for %q; set one: cme config set username <name>", p.Name)
	}

	ram := p.RAM
	if ram == 0 {
		ram = cfg.RAM
	}

	id, installed := profileTarget(p)
	if !installed {
		if noInstall || !ui.IsTerminal() {
			return notInstalledError(p)
		}
		if !ui.Confirm(fmt.Sprintf("%s is not installed. Install it now?", targetLabel(p)), true) {
			return notInstalledError(p)
		}
		id, err = installTarget(p)
		if err != nil {
			return err
		}
	}

	var jvmArgs []string
	if ram > 0 {
		jvmArgs = append(jvmArgs, fmt.Sprintf("-Xmx%dG", ram), fmt.Sprintf("-Xms%dG", ram))
	} else {
		ui.Warn("no ram set for %q; using JVM defaults", p.Name)
	}
	jvmArgs = append(jvmArgs, cfg.JVMArgs...)
	jvmArgs = append(jvmArgs, p.JVMArgs...)

	gameDir := p.GameDir
	if gameDir == "" {
		gameDir, err = store.SafeJoin(store.InstancesDir(), p.Name)
		if err != nil {
			return err
		}
	}

	return launch.Launch(launch.Options{
		VersionID: id,
		Account:   account.Offline(username),
		JVMArgs:   jvmArgs,
		GameDir:   gameDir,
		JavaPath:  cfg.JavaPath,
	})
}

func validateTarget(loader, version string) error {
	switch loader {
	case "":
		m, err := manifest.Fetch()
		if err != nil {
			return err
		}
		if m.Find(version) == nil {
			return fmt.Errorf("version %q not found, try: cme version list", version)
		}
		return nil
	case "fabric":
		_, err := meta.FabricLatestLoader(version)
		return err
	case "quilt":
		_, err := meta.QuiltLatestLoader(version)
		return err
	default:
		return fmt.Errorf("unknown loader %q", loader)
	}
}

func profileTarget(p *profile.Profile) (id string, installed bool) {
	if p.Loader == "" {
		return p.Version, versionInstalled(p.Version)
	}
	if p.LoaderVersion != "" {
		id = p.Loader + "-loader-" + p.LoaderVersion + "-" + p.Version
		return id, versionInstalled(id)
	}
	if resolved, err := resolveModdedID(p.Loader, p.Version, ""); err == nil {
		return resolved, true
	}
	return "", false
}

func installTarget(p *profile.Profile) (string, error) {
	if err := preflight.RequireOnline(); err != nil {
		return "", err
	}
	m, err := manifest.FetchFresh()
	if err != nil {
		return "", err
	}
	v := m.Find(p.Version)
	if v == nil {
		return "", fmt.Errorf("version %q not found, try: cme version list", p.Version)
	}

	progress := func(stage string, done, total int) { ui.Progress(stage, done, total) }

	if p.Loader == "" {
		ui.Info("installing %s", p.Version)
		if _, err := installer.Install(v, progress); err != nil {
			return "", err
		}
		return p.Version, nil
	}

	install := installer.InstallFabric
	if p.Loader == "quilt" {
		install = installer.InstallQuilt
	}
	ui.Info("fetching %s loader for %s", p.Loader, p.Version)
	meta, err := install(v, p.LoaderVersion, progress)
	if err != nil {
		return "", err
	}
	return meta.ID, nil
}

func versionInstalled(id string) bool {
	_, err := os.Stat(filepath.Join(store.VersionDir(id), id+".json"))
	return err == nil
}

func targetLabel(p *profile.Profile) string {
	if p.Loader == "" {
		return p.Version
	}
	if p.LoaderVersion != "" {
		return p.Loader + " " + p.Version + " " + p.LoaderVersion
	}
	return p.Loader + " " + p.Version
}

func notInstalledError(p *profile.Profile) error {
	if p.Loader == "" {
		return fmt.Errorf("%s is not installed; run: cme install %s", p.Version, p.Version)
	}
	return fmt.Errorf("%s %s is not installed; run: cme install %s %s", p.Loader, p.Version, p.Loader, p.Version)
}

func cmdConfig(args []string) error {
	if len(args) == 0 || wantsHelp(args) {
		fmt.Print(configUsage)
		return nil
	}
	switch args[0] {
	case "list":
		return configList()
	case "get":
		if len(args) != 2 {
			return fmt.Errorf("usage: cme config get <key>")
		}
		return configGet(args[1])
	case "set":
		if len(args) != 3 {
			return fmt.Errorf("usage: cme config set <key> <value>")
		}
		return configSet(args[1], args[2])
	default:
		return fmt.Errorf("unknown config command %q (try: set, get, list)", args[0])
	}
}

func configSet(key, value string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	switch key {
	case "username":
		cfg.Username = value
	case "ram":
		n, e := strconv.Atoi(value)
		if e != nil || n <= 0 {
			return fmt.Errorf("ram must be a positive integer (GB), got %q", value)
		}
		cfg.RAM = n
	case "java":
		cfg.JavaPath = value
	default:
		return fmt.Errorf("unknown key %q (known: username, ram, java)", key)
	}
	if err := config.Save(cfg); err != nil {
		return err
	}
	ui.Success("config %s = %s", key, value)
	return nil
}

func configGet(key string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	switch key {
	case "username":
		fmt.Println(cfg.Username)
	case "ram":
		if cfg.RAM > 0 {
			fmt.Println(cfg.RAM)
		} else {
			fmt.Println()
		}
	case "java":
		fmt.Println(cfg.JavaPath)
	default:
		return fmt.Errorf("unknown key %q (known: username, ram, java)", key)
	}
	return nil
}

func configList() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	tw := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintf(tw, "username\t%s\n", unsetLabel(cfg.Username))
	fmt.Fprintf(tw, "ram\t%s\n", ramConfigLabel(cfg.RAM))
	fmt.Fprintf(tw, "java\t%s\n", unsetLabel(cfg.JavaPath))
	if len(cfg.JVMArgs) > 0 {
		fmt.Fprintf(tw, "jvmArgs\t%s\n", strings.Join(cfg.JVMArgs, " "))
	}
	return tw.Flush()
}

func printProfile(p *profile.Profile, cfg *config.Config) {
	loader := "vanilla"
	if p.Loader != "" {
		loader = p.Loader
		if p.LoaderVersion != "" {
			loader += " " + p.LoaderVersion
		} else {
			loader += " (latest)"
		}
	}

	user := p.Username
	src := ""
	if user == "" && cfg.Username != "" {
		user, src = cfg.Username, " (from config)"
	}
	if user == "" {
		user = "(none set)"
	}

	ram := "(unset)"
	if p.RAM > 0 {
		ram = strconv.Itoa(p.RAM) + "G"
	} else if cfg.RAM > 0 {
		ram = strconv.Itoa(cfg.RAM) + "G (from config)"
	}

	gameDir := p.GameDir
	if gameDir == "" {
		gameDir = filepath.Join(store.InstancesDir(), p.Name)
	}

	fmt.Println(ui.Bold(p.Name))
	fmt.Printf("  %s  %s\n", ui.Dim("version "), p.Version)
	fmt.Printf("  %s  %s\n", ui.Dim("loader  "), loader)
	fmt.Printf("  %s  %s%s\n", ui.Dim("username"), user, src)
	fmt.Printf("  %s  %s\n", ui.Dim("ram     "), ram)
	fmt.Printf("  %s  %s\n", ui.Dim("game dir"), gameDir)
	if len(p.JVMArgs) > 0 {
		fmt.Printf("  %s  %s\n", ui.Dim("jvm args"), strings.Join(p.JVMArgs, " "))
	}
}

func parseTarget(args []string) (loader, version, loaderVer string, rest []string, err error) {
	if len(args) == 0 {
		return "", "", "", nil, nil
	}
	if args[0] == "fabric" || args[0] == "quilt" {
		loader = args[0]
		if len(args) < 2 {
			return "", "", "", nil, fmt.Errorf("%s needs a version", loader)
		}
		version = args[1]
		rest = args[2:]
		if len(rest) > 0 && !strings.HasPrefix(rest[0], "-") {
			loaderVer = rest[0]
			rest = rest[1:]
		}
		return loader, version, loaderVer, rest, nil
	}
	return "", args[0], "", args[1:], nil
}

func parseProfileFlags(args []string) (username string, ram int, jvmArgs []string, dir string, err error) {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--username":
			if i+1 >= len(args) {
				return "", 0, nil, "", fmt.Errorf("--username needs a value")
			}
			username = args[i+1]
			i++
		case "--ram":
			if i+1 >= len(args) {
				return "", 0, nil, "", fmt.Errorf("--ram needs a value")
			}
			n, e := strconv.Atoi(args[i+1])
			if e != nil || n <= 0 {
				return "", 0, nil, "", fmt.Errorf("--ram must be a positive integer (GB), got %q", args[i+1])
			}
			ram = n
			i++
		case "--jvm-arg":
			if i+1 >= len(args) {
				return "", 0, nil, "", fmt.Errorf("--jvm-arg needs a value")
			}
			jvmArgs = append(jvmArgs, args[i+1])
			i++
		case "--dir":
			if i+1 >= len(args) {
				return "", 0, nil, "", fmt.Errorf("--dir needs a value")
			}
			dir = args[i+1]
			i++
		default:
			return "", 0, nil, "", fmt.Errorf("unknown flag %q", args[i])
		}
	}
	return username, ram, jvmArgs, dir, nil
}

func loaderLabel(p *profile.Profile) string {
	if p.Loader == "" {
		return ""
	}
	if p.LoaderVersion != "" {
		return p.Loader + " " + p.LoaderVersion
	}
	return p.Loader
}

func ramLabel(n int) string {
	if n <= 0 {
		return "-"
	}
	return strconv.Itoa(n) + "G"
}

func ramConfigLabel(n int) string {
	if n <= 0 {
		return "(unset)"
	}
	return strconv.Itoa(n) + "G"
}

func dash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func unsetLabel(s string) string {
	if s == "" {
		return "(unset)"
	}
	return s
}
