package profile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/onceprgm/cme/internal/store"
)

const currentVersion = 1

type Profile struct {
	Name          string   `json:"name"`
	Loader        string   `json:"loader,omitempty"`
	Version       string   `json:"version"`
	LoaderVersion string   `json:"loaderVersion,omitempty"`
	Username      string   `json:"username,omitempty"`
	RAM           int      `json:"ram,omitempty"`
	JVMArgs       []string `json:"jvmArgs,omitempty"`
	GameDir       string   `json:"gameDir,omitempty"`
}

type registry struct {
	ConfigVersion int                 `json:"configVersion"`
	Profiles      map[string]*Profile `json:"profiles"`
}

func path() string {
	return filepath.Join(store.ConfigDir(), "profiles.json")
}

func load() (*registry, error) {
	raw, err := os.ReadFile(path())
	if err != nil {
		if os.IsNotExist(err) {
			return &registry{ConfigVersion: currentVersion, Profiles: map[string]*Profile{}}, nil
		}
		return nil, err
	}
	var r registry
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("parse profiles: %w", err)
	}
	if r.Profiles == nil {
		r.Profiles = map[string]*Profile{}
	}
	return &r, nil
}

func persist(r *registry) error {
	r.ConfigVersion = currentVersion
	if err := store.Ensure(store.ConfigDir()); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	dst := path()
	tmp := dst + ".part"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, dst)
}

func List() ([]*Profile, error) {
	r, err := load()
	if err != nil {
		return nil, err
	}
	out := make([]*Profile, 0, len(r.Profiles))
	for _, p := range r.Profiles {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func Get(name string) (*Profile, error) {
	r, err := load()
	if err != nil {
		return nil, err
	}
	p, ok := r.Profiles[name]
	if !ok {
		return nil, fmt.Errorf("profile %q not found (see: cme profile list)", name)
	}
	return p, nil
}

func Exists(name string) bool {
	r, err := load()
	return err == nil && r.Profiles[name] != nil
}

func Save(p *Profile) error {
	r, err := load()
	if err != nil {
		return err
	}
	r.Profiles[p.Name] = p
	return persist(r)
}

func Delete(name string) error {
	r, err := load()
	if err != nil {
		return err
	}
	if r.Profiles[name] == nil {
		return fmt.Errorf("profile %q not found", name)
	}
	delete(r.Profiles, name)
	return persist(r)
}
