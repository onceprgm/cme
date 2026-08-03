package manifest

func Merge(parent, child *VersionMeta) *VersionMeta {
	out := *parent
	out.ID = firstNonEmpty(child.ID, parent.ID)
	out.MainClass = firstNonEmpty(child.MainClass, parent.MainClass)
	out.Type = firstNonEmpty(child.Type, parent.Type)
	out.InheritsFrom = ""
	out.Libraries = mergeLibraries(parent.Libraries, child.Libraries)
	out.Arguments = mergeArguments(parent.Arguments, child.Arguments)
	out.MinecraftArguments = mergeMinecraftArgs(parent.MinecraftArguments, child.MinecraftArguments)
	if child.JavaVersion.MajorVersion > out.JavaVersion.MajorVersion {
		out.JavaVersion = child.JavaVersion
	}
	return &out
}

func mergeLibraries(parent, child []Library) []Library {
	childKeys := map[string]bool{}
	for _, l := range child {
		if key := l.MavenKey(); key != "" {
			childKeys[key] = true
		}
	}

	out := make([]Library, 0, len(parent)+len(child))
	out = append(out, child...)
	for _, l := range parent {
		if key := l.MavenKey(); key != "" && childKeys[key] {
			continue
		}
		out = append(out, l)
	}
	return out
}

func mergeArguments(parent, child *Arguments) *Arguments {
	if child == nil {
		return parent
	}
	if parent == nil {
		return child
	}
	out := &Arguments{}
	out.Game = append(out.Game, parent.Game...)
	out.Game = append(out.Game, child.Game...)
	out.JVM = append(out.JVM, parent.JVM...)
	out.JVM = append(out.JVM, child.JVM...)
	return out
}

func mergeMinecraftArgs(parent, child string) string {
	switch {
	case parent == "":
		return child
	case child == "":
		return parent
	default:
		return parent + " " + child
	}
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
