# cme.sh

Minimal Minecraft launcher for Linux. Fast. Offline-ready. No bloat.

`cme` is a command-line Minecraft launcher with zero dependencies - the entire
thing is built on the Go standard library. It installs and launches vanilla
Minecraft in offline mode, with parallel downloads, SHA-1 verification, and
XDG-compliant file layout. No GUI, no Electron, no runtime deps.

> **Status: alpha (`0.1.0-alpha`).** Installing and launching work. Account
> system is offline-only. Linux x86_64 is the only tested platform.

## Requirements

- Linux (x86_64; ARM is untested)
- [Go](https://golang.org) 1.22+ to build
- A Java runtime matching the version you want to play
  (Java 8 for old versions, 17 for 1.18–1.20, 21 for 1.20.5+).
  `cme` finds Java in your `PATH` or `/usr/lib/jvm`. Automatic install is planned.

## Install

> Binary releases are coming. For now, build from source:

```sh
git clone https://github.com/onceprgm/cme
cd cme
go build -o cme ./cmd/cme
```

## Usage

```sh
# list versions (filter by type)
cme version list
cme version list --release
cme version list --snapshot
cme version list --old-beta
cme version list --old-alpha

# install a version: client JAR + libraries + natives + assets, all SHA-1 verified
cme install 1.20.1

# launch in offline mode
cme launch 1.20.1 --username Steve
cme launch 1.20.1 --username Steve --ram 4
```

`--ram` is in gigabytes and sets both `-Xmx` and `-Xms`. The offline UUID is
derived deterministically from the username, so it stays consistent across
sessions and matches what a server computes for the same name.

### Output
 
```
* 26.2-rc-1                  snapshot   2026-06-11
  26.2-pre-6                 snapshot   2026-06-10
  26.2-pre-5                 snapshot   2026-06-08
  26.2-pre-4                 snapshot   2026-06-04
  ...
* 26.1.2                     release    2026-04-09
  ...
  rd-132328                  old_alpha  2009-05-13
  rd-132211                  old_alpha  2009-05-13
```
 
`*` marks the latest release or snapshot.

### Version support

| Versions        | Status                                           |
|-----------------|--------------------------------------------------|
| 1.9 and newer   | Fully supported                                  |
| 1.7.3 – 1.8.x   | Launches, but sound and languages may be missing¹   |
| Older than 1.7.3| Not yet supported¹                               |

¹ These use the legacy asset layout, which isn't implemented yet (planned).

## File layout

`cme` follows the XDG Base Directory spec:

```
~/.local/share/cme/
  versions/<id>/      version JSON, client JAR, extracted natives
  libraries/          shared across versions
  assets/             shared across versions
  instances/<id>/     per-version game directory (saves, logs, options)
```

## Roadmap

- [x] fetch and parse the version manifest
- [x] install vanilla versions (client, libraries, natives, assets - SHA-1 verified)
- [x] parallel downloads with retry and resume-by-hash
- [x] launch installed versions in offline mode
- [ ] Fabric and Quilt
- [ ] integrity check (`cme verify`)
- [ ] automatic Java installation
- [ ] profiles and config file
- [ ] Forge / NeoForge
- [ ] legacy asset layout (sound for pre-1.9 versions)

## License

[MIT](LICENSE)
