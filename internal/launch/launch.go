package launch

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/onceprgm/cme/internal/account"
	"github.com/onceprgm/cme/internal/clog"
	"github.com/onceprgm/cme/internal/java"
	"github.com/onceprgm/cme/internal/manifest"
	"github.com/onceprgm/cme/internal/store"
	"github.com/onceprgm/cme/internal/ui"
)

const (
	launcherName          = "cme"
	launcherVersion       = "0.1.2-alpha"
	bootstrapLauncherMain = "cpw.mods.bootstraplauncher.BootstrapLauncher"
)

var offlineAuthArgs = []string{
	"-Dminecraft.api.env=custom",
	"-Dminecraft.api.auth.host=https://invalid.invalid",
	"-Dminecraft.api.account.host=https://invalid.invalid",
	"-Dminecraft.api.session.host=https://invalid.invalid",
	"-Dminecraft.api.services.host=https://invalid.invalid",
}

type Options struct {
	VersionID string
	Account   account.Account
	GameDir   string
	JavaPath  string
	JVMArgs   []string
	Quiet     bool
}

func Launch(opts Options) error {
	start := time.Now()
	versionDir := store.VersionDir(opts.VersionID)
	meta, err := manifest.LoadVersionMeta(filepath.Join(versionDir, opts.VersionID+".json"))
	if err != nil {
		return fmt.Errorf("load version %s: %w (is it installed?)", opts.VersionID, err)
	}

	clog.Info("launch", "version", opts.VersionID, "user", opts.Account.Username, "uuid", opts.Account.UUID)
	clog.Debug("display env",
		"WAYLAND_DISPLAY", os.Getenv("WAYLAND_DISPLAY"),
		"DISPLAY", os.Getenv("DISPLAY"),
		"XDG_RUNTIME_DIR", os.Getenv("XDG_RUNTIME_DIR"))

	clientID := opts.VersionID
	if meta.InheritsFrom != "" {
		parentID := meta.InheritsFrom
		parent, perr := manifest.LoadVersionMeta(filepath.Join(store.VersionDir(parentID), parentID+".json"))
		if perr != nil {
			return fmt.Errorf("load parent version %s: %w (install it first)", parentID, perr)
		}
		meta = manifest.Merge(parent, meta)
		clientID = parentID
		clog.Debug("resolved inheritance", "id", opts.VersionID, "parent", parentID)
	}
	clientDir := store.VersionDir(clientID)

	javaBin, err := java.Resolve(meta.JavaVersion.MajorVersion, opts.JavaPath)
	if err != nil {
		return err
	}
	clog.Debug("resolved java", "path", javaBin, "want_major", meta.JavaVersion.MajorVersion)

	ctx := manifest.CurrentContext()

	var cp []string
	for _, p := range meta.ClasspathPaths(ctx) {
		cp = append(cp, filepath.Join(store.LibrariesDir(), filepath.FromSlash(p)))
	}
	if meta.MainClass != bootstrapLauncherMain {
		cp = append(cp, filepath.Join(clientDir, clientID+".jar"))
	}
	classpath := strings.Join(cp, string(os.PathListSeparator))

	gameDir := opts.GameDir
	if gameDir == "" {
		gameDir = filepath.Join(store.InstancesDir(), opts.VersionID)
	}
	if err := store.Ensure(gameDir); err != nil {
		return err
	}

	nativesDir := filepath.Join(clientDir, "natives")

	vars := map[string]string{
		"auth_player_name":    opts.Account.Username,
		"auth_uuid":           opts.Account.UUID,
		"auth_access_token":   opts.Account.AccessToken,
		"user_type":           opts.Account.UserType,
		"version_name":        opts.VersionID,
		"version_type":        versionType(meta.Type),
		"game_directory":      gameDir,
		"assets_root":         store.AssetsDir(),
		"assets_index_name":   meta.AssetIndex.ID,
		"classpath":           classpath,
		"classpath_separator": string(os.PathListSeparator),
		"library_directory":   store.LibrariesDir(),
		"natives_directory":   nativesDir,
		"launcher_name":       launcherName,
		"launcher_version":    launcherVersion,
		"clientid":            "",
		"auth_xuid":           "",
		"user_properties":     "{}",
		"game_assets":         store.AssetsDir(),
		"auth_session":        "token:" + opts.Account.AccessToken + ":" + opts.Account.UUID,
	}

	args := meta.JVMArgs(ctx, vars)
	if opts.Account.Offline {
		args = append(args, offlineAuthArgs...)
	}
	args = append(args, opts.JVMArgs...)
	args = append(args, meta.MainClass)
	args = append(args, meta.GameArgs(ctx, vars)...)

	clog.Debug("classpath", "entries", len(cp))
	clog.Debug("game dir", "path", gameDir)
	clog.Debug("natives dir", "path", nativesDir)
	clog.Debug("launch command", "java", javaBin, "args", strings.Join(args, " "))

	logDir := filepath.Join(gameDir, "logs")
	if err := store.Ensure(logDir); err != nil {
		return err
	}
	logFile, err := os.Create(filepath.Join(logDir, "cme-latest.log"))
	if err != nil {
		return err
	}
	defer logFile.Close()

	cmd := exec.Command(javaBin, args...)
	cmd.Dir = gameDir
	if opts.Quiet {
		cmd.Stdout = logFile
		cmd.Stderr = logFile
	} else {
		cmd.Stdout = io.MultiWriter(os.Stdout, logFile)
		cmd.Stderr = io.MultiWriter(os.Stderr, logFile)
	}

	ui.Info("launching %s as %s (java: %s)", opts.VersionID, opts.Account.Username, javaBin)
	ui.Success("ready in %s", time.Since(start).Round(time.Millisecond))
	if err := cmd.Run(); err != nil {
		clog.Error("minecraft exited", "version", opts.VersionID, "err", err.Error())
		return fmt.Errorf("minecraft exited: %w", err)
	}
	clog.Info("minecraft exited cleanly", "version", opts.VersionID)
	return nil
}

func versionType(t string) string {
	if t == "" {
		return "release"
	}
	return t
}
