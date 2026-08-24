package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/seanho/gitid/internal/profile"
)

type Config struct {
	Profiles []profile.Profile `json:"profiles"`
}

func Path(home string) string {
	return filepath.Join(home, ".config", "gitid", "config.json")
}

func Load(home string) (Config, error) {
	path := Path(home)
	contents, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, err
	}
	var config Config
	if err := json.Unmarshal(contents, &config); err != nil {
		return Config{}, fmt.Errorf("parse %s: %w", path, err)
	}
	return config, nil
}

func Save(home string, config Config) error {
	directory := filepath.Dir(Path(home))
	if err := os.MkdirAll(directory, 0700); err != nil {
		return err
	}
	contents, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	contents = append(contents, '\n')
	return atomicWrite(Path(home), contents, 0600)
}

func (c Config) Find(name string) (profile.Profile, bool) {
	for _, item := range c.Profiles {
		if item.Name == name {
			return item, true
		}
	}
	return profile.Profile{}, false
}

func (c Config) Add(item profile.Profile) (Config, error) {
	if _, found := c.Find(item.Name); found {
		return c, fmt.Errorf("profile %q already exists", item.Name)
	}
	for _, existing := range c.Profiles {
		if item.HostAlias != "" && existing.HostAlias == item.HostAlias {
			return c, fmt.Errorf("SSH host alias %q is already used by profile %q", item.HostAlias, existing.Name)
		}
	}
	c.Profiles = append(c.Profiles, item)
	return c, nil
}

func (c Config) Remove(name string) (Config, error) {
	for index, item := range c.Profiles {
		if item.Name == name {
			c.Profiles = append(c.Profiles[:index], c.Profiles[index+1:]...)
			return c, nil
		}
	}
	return c, fmt.Errorf("profile %q does not exist", name)
}

func atomicWrite(path string, contents []byte, mode os.FileMode) error {
	directory := filepath.Dir(path)
	temporary, err := os.CreateTemp(directory, ".gitid-*")
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
