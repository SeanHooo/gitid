package sshconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/seanhooo/gitid/internal/profile"
)

const startMarker = "# >>> gitid managed block >>>"
const endMarker = "# <<< gitid managed block <<<"

func Path(home string) string {
	return filepath.Join(home, ".ssh", "config")
}

func Ensure(home string, profiles []profile.Profile) error {
	directory := filepath.Dir(Path(home))
	if err := os.MkdirAll(directory, 0700); err != nil {
		return err
	}
	path := Path(home)
	contents, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	updated, err := Render(string(contents), profiles)
	if err != nil {
		return err
	}
	return atomicWrite(path, []byte(updated), 0600)
}

func Render(existing string, profiles []profile.Profile) (string, error) {
	start := strings.Index(existing, startMarker)
	end := strings.Index(existing, endMarker)
	if (start == -1) != (end == -1) || (start != -1 && end < start) {
		return "", fmt.Errorf("SSH config contains malformed gitid managed block")
	}
	unmanaged := existing
	if start != -1 {
		end += len(endMarker)
		unmanaged = existing[:start] + existing[end:]
	}
	for _, item := range profiles {
		if containsHost(unmanaged, item.HostAlias) {
			return "", fmt.Errorf("SSH config already defines unmanaged Host %q", item.HostAlias)
		}
	}
	block := buildBlock(profiles)
	unmanaged = strings.TrimRight(unmanaged, "\n")
	if unmanaged == "" {
		return block, nil
	}
	return unmanaged + "\n\n" + block, nil
}

func HasProfileHost(home string, item profile.Profile) (bool, error) {
	contents, err := os.ReadFile(Path(home))
	if err != nil {
		return false, err
	}
	return strings.Contains(string(contents), "Host "+item.HostAlias+"\n") && strings.Contains(string(contents), "IdentityFile "+item.ExpandedKeyPath()+"\n"), nil
}

func buildBlock(profiles []profile.Profile) string {
	var builder strings.Builder
	builder.WriteString(startMarker + "\n")
	for _, item := range profiles {
		fmt.Fprintf(&builder, "Host %s\n", item.HostAlias)
		fmt.Fprintf(&builder, "  HostName %s\n", item.HostName)
		builder.WriteString("  User git\n")
		fmt.Fprintf(&builder, "  IdentityFile %s\n", item.ExpandedKeyPath())
		builder.WriteString("  IdentitiesOnly yes\n\n")
	}
	builder.WriteString(endMarker + "\n")
	return builder.String()
}

func containsHost(contents, alias string) bool {
	for _, line := range strings.Split(contents, "\n") {
		fields := strings.Fields(line)
		if len(fields) > 1 && strings.EqualFold(fields[0], "Host") {
			for _, host := range fields[1:] {
				if host == alias {
					return true
				}
			}
		}
	}
	return false
}

func atomicWrite(path string, contents []byte, mode os.FileMode) error {
	temporary, err := os.CreateTemp(filepath.Dir(path), ".gitid-*")
	if err != nil {
		return err
	}
	name := temporary.Name()
	defer os.Remove(name)
	if err := temporary.Chmod(mode); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(contents); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}
