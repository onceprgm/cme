package manifest

import "testing"

func TestMavenPath(t *testing.T) {
	cases := map[string]string{
		"net.fabricmc:fabric-loader:0.16.9":         "net/fabricmc/fabric-loader/0.16.9/fabric-loader-0.16.9.jar",
		"org.ow2.asm:asm:9.7":                       "org/ow2/asm/asm/9.7/asm-9.7.jar",
		"io.netty:netty-transport:4.1:linux-x86_64": "io/netty/netty-transport/4.1/netty-transport-4.1-linux-x86_64.jar",
		"bad:name": "",
		"":         "",
	}
	for name, want := range cases {
		if got := mavenPath(name); got != want {
			t.Errorf("mavenPath(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestArtifactFromMaven(t *testing.T) {
	l := Library{Name: "net.fabricmc:fabric-loader:0.16.9", URL: "https://maven.fabricmc.net/"}
	f, ok := l.Artifact()
	if !ok {
		t.Fatal("expected artifact from maven name")
	}
	if f.Path != "net/fabricmc/fabric-loader/0.16.9/fabric-loader-0.16.9.jar" {
		t.Errorf("path = %q", f.Path)
	}
	if f.URL != "https://maven.fabricmc.net/net/fabricmc/fabric-loader/0.16.9/fabric-loader-0.16.9.jar" {
		t.Errorf("url = %q", f.URL)
	}
}

func TestArtifactPrefersDownloads(t *testing.T) {
	var l Library
	l.Name = "com.example:thing:1.0"
	l.Downloads.Artifact = &LibFile{Path: "explicit/path.jar", URL: "https://cdn/path.jar", SHA1: "abc"}
	f, ok := l.Artifact()
	if !ok || f.Path != "explicit/path.jar" || f.SHA1 != "abc" {
		t.Errorf("expected explicit downloads artifact, got %+v ok=%v", f, ok)
	}
}

func TestMergeChildWins(t *testing.T) {
	parent := &VersionMeta{
		ID:        "1.21.4",
		Type:      "release",
		MainClass: "net.minecraft.client.main.Main",
		Libraries: []Library{
			{Name: "org.ow2.asm:asm:9.6"},
			{Name: "com.google.code.gson:gson:2.10"},
		},
	}
	parent.JavaVersion.MajorVersion = 21
	child := &VersionMeta{
		ID:           "fabric-loader-0.16.9-1.21.4",
		InheritsFrom: "1.21.4",
		MainClass:    "net.fabricmc.loader.impl.launch.knot.KnotClient",
		Libraries: []Library{
			{Name: "net.fabricmc:fabric-loader:0.16.9", URL: "https://maven.fabricmc.net/"},
			{Name: "org.ow2.asm:asm:9.7", URL: "https://maven.fabricmc.net/"},
		},
	}

	out := Merge(parent, child)

	if out.ID != "fabric-loader-0.16.9-1.21.4" {
		t.Errorf("id = %q", out.ID)
	}
	if out.MainClass != "net.fabricmc.loader.impl.launch.knot.KnotClient" {
		t.Errorf("mainClass = %q", out.MainClass)
	}
	if out.InheritsFrom != "" {
		t.Errorf("merged result should not inherit, got %q", out.InheritsFrom)
	}
	if out.JavaVersion.MajorVersion != 21 {
		t.Errorf("java = %d", out.JavaVersion.MajorVersion)
	}

	asm := ""
	count := map[string]int{}
	for _, l := range out.Libraries {
		count[l.MavenKey()]++
		if l.MavenKey() == "org.ow2.asm:asm" {
			asm = l.Name
		}
	}
	if count["org.ow2.asm:asm"] != 1 {
		t.Errorf("asm should appear once, got %d", count["org.ow2.asm:asm"])
	}
	if asm != "org.ow2.asm:asm:9.7" {
		t.Errorf("child asm version should win, got %q", asm)
	}
	if count["net.fabricmc:fabric-loader"] != 1 || count["com.google.code.gson:gson"] != 1 {
		t.Errorf("expected loader and gson present once: %v", count)
	}
}
