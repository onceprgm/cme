package profile

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"sync"
	"testing"
)

func isolate(t *testing.T) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, ".config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, ".local", "share"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(root, ".cache"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, ".local", "state"))
}

func TestAddJVMArg_Appends(t *testing.T) {
	p := &Profile{}
	if !p.AddJVMArg("-XX:+UseG1GC") {
		t.Fatal("AddJVMArg(-XX:+UseG1GC) = false, want true")
	}
	if !p.AddJVMArg("-Xmx4G") {
		t.Fatal("AddJVMArg(-Xmx4G) = false, want true")
	}
	if want := []string{"-XX:+UseG1GC", "-Xmx4G"}; !slices.Equal(p.JVMArgs, want) {
		t.Fatalf("JVMArgs = %v, want %v", p.JVMArgs, want)
	}
}

func TestAddJVMArg_Deduplicates(t *testing.T) {
	p := &Profile{}
	p.AddJVMArg("-XX:+UseZGC")
	if p.AddJVMArg("-XX:+UseZGC") {
		t.Fatal("duplicate add = true, want false")
	}
	if len(p.JVMArgs) != 1 {
		t.Fatalf("len(JVMArgs) = %d, want 1", len(p.JVMArgs))
	}
}

func TestAddJVMArg_EmptyRejected(t *testing.T) {
	p := &Profile{}
	for _, arg := range []string{"", "   ", "\t"} {
		if p.AddJVMArg(arg) {
			t.Fatalf("AddJVMArg(%q) = true, want false", arg)
		}
	}
	if len(p.JVMArgs) != 0 {
		t.Fatalf("len(JVMArgs) = %d, want 0", len(p.JVMArgs))
	}
}

func TestRemoveJVMArg_Exists(t *testing.T) {
	p := &Profile{JVMArgs: []string{"-a", "-b", "-c"}}
	if !p.RemoveJVMArg("-b") {
		t.Fatal("RemoveJVMArg(-b) = false, want true")
	}
	if want := []string{"-a", "-c"}; !slices.Equal(p.JVMArgs, want) {
		t.Fatalf("JVMArgs = %v, want %v", p.JVMArgs, want)
	}
}

func TestRemoveJVMArg_Missing(t *testing.T) {
	p := &Profile{JVMArgs: []string{"-a"}}
	if p.RemoveJVMArg("-zzz") {
		t.Fatal("RemoveJVMArg(-zzz) = true, want false")
	}
	if want := []string{"-a"}; !slices.Equal(p.JVMArgs, want) {
		t.Fatalf("JVMArgs = %v, want %v", p.JVMArgs, want)
	}
}

func TestRemoveJVMArg_MultipleOccurrencesRemovesFirst(t *testing.T) {
	p := &Profile{JVMArgs: []string{"-x", "-y", "-x"}}
	if !p.RemoveJVMArg("-x") {
		t.Fatal("remove = false, want true")
	}
	if want := []string{"-y", "-x"}; !slices.Equal(p.JVMArgs, want) {
		t.Fatalf("JVMArgs = %v, want %v", p.JVMArgs, want)
	}
}

func TestSetJVMArgs_Replaces(t *testing.T) {
	p := &Profile{JVMArgs: []string{"-old"}}
	src := []string{"-a", "-b"}
	p.SetJVMArgs(src)
	src[0] = "-mutated"
	if want := []string{"-a", "-b"}; !slices.Equal(p.JVMArgs, want) {
		t.Fatalf("JVMArgs = %v, want %v", p.JVMArgs, want)
	}
}

func TestSetJVMArgs_NilClears(t *testing.T) {
	p := &Profile{JVMArgs: []string{"-a"}}
	p.SetJVMArgs(nil)
	if len(p.JVMArgs) != 0 {
		t.Fatalf("len(JVMArgs) = %d, want 0", len(p.JVMArgs))
	}
}

func TestJVMArgsCopy_DoesNotAlias(t *testing.T) {
	p := &Profile{JVMArgs: []string{"-a"}}
	cp := p.JVMArgsCopy()
	cp[0] = "-mutated"
	if p.JVMArgs[0] != "-a" {
		t.Fatalf("profile mutated through copy: %v", p.JVMArgs)
	}
}

func TestUpdate_Persists(t *testing.T) {
	isolate(t)
	if err := Save(&Profile{Name: "p1", Version: "1.20.1"}); err != nil {
		t.Fatal(err)
	}
	err := Update("p1", func(p *Profile) error {
		p.AddJVMArg("-XX:+UseZGC")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := Get("p1")
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"-XX:+UseZGC"}; !slices.Equal(got.JVMArgs, want) {
		t.Fatalf("JVMArgs = %v, want %v", got.JVMArgs, want)
	}
}

func TestUpdate_ErrorRollsBack(t *testing.T) {
	isolate(t)
	if err := Save(&Profile{Name: "p1", Version: "1.20.1", JVMArgs: []string{"-keep"}}); err != nil {
		t.Fatal(err)
	}
	boom := errors.New("boom")
	err := Update("p1", func(p *Profile) error {
		p.AddJVMArg("-drop")
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("Update err = %v, want %v", err, boom)
	}
	got, err := Get("p1")
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"-keep"}; !slices.Equal(got.JVMArgs, want) {
		t.Fatalf("JVMArgs = %v, want %v", got.JVMArgs, want)
	}
}

func TestUpdate_NotFound(t *testing.T) {
	isolate(t)
	if err := Update("missing", func(*Profile) error { return nil }); err == nil {
		t.Fatal("Update(missing) = nil, want error")
	}
}

func TestUpdate_ConcurrentSafety(t *testing.T) {
	isolate(t)
	if err := Save(&Profile{Name: "p1", Version: "1.20.1"}); err != nil {
		t.Fatal(err)
	}
	const n = 20
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if err := Update("p1", func(p *Profile) error {
				p.AddJVMArg(fmt.Sprintf("-Darg.%d=%d", i, i))
				return nil
			}); err != nil {
				t.Error(err)
			}
		}(i)
	}
	wg.Wait()
	got, err := Get("p1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got.JVMArgs) != n {
		t.Fatalf("len(JVMArgs) = %d, want %d (lost updates?)", len(got.JVMArgs), n)
	}
}
