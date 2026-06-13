# cme.sh
 
Minimal Minecraft launcher for Linux. Fast. Offline-ready. No bloat.
 
## Install
 
> Work in progress. Binary releases coming soon. >.<
 
```sh
git clone https://github.com/onceprgm/cme
cd cme
go build -o cme ./cmd/cme
```
 
## Usage
 
```sh
# list all versions
cme version list
 
# filter by type
cme version list --release
cme version list --snapshot
cme version list --old-beta
cme version list --old-alpha
 
# install a version (client JAR + libraries + natives + assets, SHA-1 verified)
cme install 1.21.4
cme install 26.1.2
```
 
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
 
## Roadmap
 
- [x] fetch and parse version manifest
- [x] filter versions by type
- [x] install vanilla version (client JAR + version JSON, SHA-1 verified)
- [x] download libraries and extract natives (parallel, platform-filtered)
- [x] download asset index and objects (deduplicated by hash, parallel)
- [ ] launch installed version

## Stack
 
- [Go](https://golang.org) 1.26.4

## License
 
[MIT](LICENSE)
