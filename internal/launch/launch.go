package launch

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/onceprgm/cme/internal/account"
	"github.com/onceprgm/cme/internal/java"
	"github.com/onceprgm/cme/internal/manifest"
	"github.com/onceprgm/cme/internal/store"
)

const (
	launcherName    = "cme"
	launcherVersion = "0.1.0-alpha"
)

type Options struct {
	VersionID string
	Account   account.Account
	GameDir   string
	JavaPath  string
	JVMArgs   []string
}

func Launch(opts Options) error {
	versionDir := store.VersionDir(opts.VersionID)
	meta, err := manifest.LoadVersionMeta(filepath.Join(versionDir, opts.VersionID+".json"))
	if err != nil {
		return fmt.Errorf("load version %s: %w (is it installed?)", opts.VersionID, err)
	}

	javaBin, err := java.Resolve(meta.JavaVersion.MajorVersion, opts.JavaPath)
	if err != nil {
		return err
	}

	ctx := manifest.CurrentContext()

	var cp []string
	for _, p := range meta.ClasspathPaths(ctx) {
		cp = append(cp, filepath.Join(store.LibrariesDir(), filepath.FromSlash(p)))
	}
	cp = append(cp, filepath.Join(versionDir, opts.VersionID+".jar"))
	classpath := strings.Join(cp, string(os.PathListSeparator))

	gameDir := opts.GameDir
	if gameDir == "" {
		gameDir = filepath.Join(store.InstancesDir(), opts.VersionID)
	}
	if err := store.Ensure(gameDir); err != nil {
		return err
	}

	nativesDir := filepath.Join(versionDir, "natives")

	vars := map[string]string{
		"auth_player_name":  opts.Account.Username,
		"auth_uuid":         opts.Account.UUID,
		"auth_access_token": opts.Account.AccessToken,
		"user_type":         opts.Account.UserType,
		"version_name":      opts.VersionID,
		"version_type":      "release",
		"game_directory":    gameDir,
		"assets_root":       store.AssetsDir(),
		"assets_index_name": meta.AssetIndex.ID,
		"classpath":         classpath,
		"natives_directory": nativesDir,
		"launcher_name":     launcherName,
		"launcher_version":  launcherVersion,
		"clientid":          "",
		"auth_xuid":         "",
		"user_properties":   "{}",
		"game_assets":       store.AssetsDir(),
		"auth_session":      "token:" + opts.Account.AccessToken + ":" + opts.Account.UUID,
	}

	args := meta.JVMArgs(ctx, vars)
	args = append(args, opts.JVMArgs...)
	args = append(args, meta.MainClass)
	args = append(args, meta.GameArgs(ctx, vars)...)

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
	cmd.Stdout = io.MultiWriter(os.Stdout, logFile)
	cmd.Stderr = io.MultiWriter(os.Stderr, logFile)

	fmt.Fprintf(os.Stderr, "launching %s as %s (java: %s)\n", opts.VersionID, opts.Account.Username, javaBin)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("minecraft exited: %w", err)
	}
	return nil
}
