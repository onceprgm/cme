Scriptability polish and a skin fix on top of 0.1.2-alpha. No breaking changes.

### Added
- `--json` on `version list`, `profile list` and `java list` for scripting.
- Stable exit codes: `0` ok, `1` error, `2` bad arguments.
- `--no-output` on `launch` and `run` keeps the game log out of the console (it is still written to `cme.log`).
- Launch prints how long the launcher took to hand off to the JVM ("ready in Nms").
- `cme open` opens the data directory, or a profile's game directory, in the file manager.

### Fixed
- The offline auth-host override that lets offline accounts join `online-mode=false` servers is now scoped to 1.16.4 and 1.16.5, so it no longer blocks skins (SkinRestorer and the session server) on other versions.
