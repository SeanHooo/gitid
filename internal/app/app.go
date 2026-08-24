package app

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/seanho/gitid/internal/config"
	"github.com/seanho/gitid/internal/gh"
	"github.com/seanho/gitid/internal/profile"
	"github.com/seanho/gitid/internal/repo"
	"github.com/seanho/gitid/internal/sshconfig"
)

type App struct {
	Home   string
	Input  io.Reader
	Output io.Writer
	Repo   repo.Repository
}

func New() (App, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return App{}, err
	}
	return App{Home: home, Input: os.Stdin, Output: os.Stdout, Repo: repo.New()}, nil
}

func (a App) Run(args []string) error {
	if len(args) == 0 {
		a.usage()
		return nil
	}
	switch args[0] {
	case "account":
		return a.account(args[1:])
	case "use":
		return a.use(args[1:])
	case "status":
		return a.status()
	case "doctor":
		return a.doctor()
	case "restore":
		return a.restore(args[1:])
	case "help", "--help", "-h":
		a.usage()
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func (a App) account(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: gitid account <add|list|remove>")
	}
	switch args[0] {
	case "add":
		return a.accountAdd(args[1:])
	case "list":
		settings, err := config.Load(a.Home)
		if err != nil {
			return err
		}
		sort.Slice(settings.Profiles, func(i, j int) bool { return settings.Profiles[i].Name < settings.Profiles[j].Name })
		for _, item := range settings.Profiles {
			fmt.Fprintf(a.Output, "%s\t%s <%s>\t%s\t%s\n", item.Name, item.GitName, item.Email, item.HostAlias, item.GitHubUser)
		}
		return nil
	case "remove":
		return a.accountRemove(args[1:])
	default:
		return fmt.Errorf("unknown account command %q", args[0])
	}
}

func (a App) accountAdd(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: gitid account add <name> --git-name NAME --email EMAIL --key PATH --host-alias ALIAS")
	}
	name := args[0]
	flags := flag.NewFlagSet("account add", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	gitName := flags.String("git-name", "", "")
	email := flags.String("email", "", "")
	githubUser := flags.String("github-user", "", "")
	key := flags.String("key", "", "")
	alias := flags.String("host-alias", "", "")
	host := flags.String("host", "github.com", "")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("usage: gitid account add <name> --git-name NAME --email EMAIL --key PATH --host-alias ALIAS")
	}
	item := profile.Profile{Name: name, GitName: *gitName, Email: *email, GitHubUser: *githubUser, KeyPath: *key, HostAlias: *alias, HostName: *host}
	item.KeyPath = item.ExpandedKeyPath()
	if err := item.Validate(); err != nil {
		return err
	}
	if item.KeyPath != "" || item.HostAlias != "" {
		if err := item.ValidateSSH(); err != nil {
			return err
		}
	}
	settings, err := config.Load(a.Home)
	if err != nil {
		return err
	}
	settings, err = settings.Add(item)
	if err != nil {
		return err
	}
	if err := sshconfig.Ensure(a.Home, settings.Profiles); err != nil {
		return err
	}
	if err := config.Save(a.Home, settings); err != nil {
		return err
	}
	fmt.Fprintf(a.Output, "Added profile %q.\n", item.Name)
	return nil
}

func (a App) accountRemove(args []string) error {
	flags := flag.NewFlagSet("account remove", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	yes := flags.Bool("yes", false, "")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 1 {
		return errors.New("usage: gitid account remove <name> [--yes]")
	}
	settings, err := config.Load(a.Home)
	if err != nil {
		return err
	}
	if _, found := settings.Find(flags.Arg(0)); !found {
		return fmt.Errorf("profile %q does not exist", flags.Arg(0))
	}
	if !*yes && !a.confirm(fmt.Sprintf("Remove profile %q?", flags.Arg(0))) {
		return nil
	}
	settings, err = settings.Remove(flags.Arg(0))
	if err != nil {
		return err
	}
	if err := sshconfig.Ensure(a.Home, settings.Profiles); err != nil {
		return err
	}
	if err := config.Save(a.Home, settings); err != nil {
		return err
	}
	fmt.Fprintf(a.Output, "Removed profile %q.\n", flags.Arg(0))
	return nil
}

func (a App) use(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: gitid use <profile> [--yes]")
	}
	name := args[0]
	flags := flag.NewFlagSet("use", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	protocol := flags.String("protocol", "ssh", "")
	yes := flags.Bool("yes", false, "")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("usage: gitid use <profile> [--protocol ssh|https] [--yes]")
	}
	if *protocol != "ssh" && *protocol != "https" {
		return errors.New("protocol must be ssh or https")
	}
	settings, err := config.Load(a.Home)
	if err != nil {
		return err
	}
	item, found := settings.Find(name)
	if !found {
		return fmt.Errorf("profile %q does not exist", name)
	}
	if *protocol == "https" {
		if err := item.ValidateHTTPS(); err != nil {
			return err
		}
	} else if err := item.ValidateSSH(); err != nil {
		return err
	}
	state, err := a.Repo.State()
	if err != nil {
		return err
	}
	parsed, err := repo.ParseRemote(state.Origin)
	if err != nil {
		return err
	}
	newRemote := parsed.SSHURL(item.HostAlias)
	if *protocol == "https" {
		newRemote = parsed.HTTPSURL()
	}
	fmt.Fprintf(a.Output, "Repository: %s\nProfile:    %s <%s>\nProtocol:   %s\nOrigin:     %s\n", state.Root, item.GitName, item.Email, *protocol, newRemote)
	if !*yes && !a.confirm("Apply these changes?") {
		return nil
	}
	if *protocol == "https" {
		if err := gh.New().CheckAndSwitch(parsed.Host, item.GitHubUser); err != nil {
			return err
		}
	} else if err := sshconfig.Ensure(a.Home, settings.Profiles); err != nil {
		return err
	}
	if err := a.Repo.Backup(state); err != nil {
		return err
	}
	if err := a.Repo.Use(item.Name, item.GitName, item.Email, newRemote); err != nil {
		return err
	}
	if *protocol == "https" {
		fmt.Fprintf(a.Output, "Switched this repository to HTTPS profile %q.\n", item.Name)
	} else {
		fmt.Fprintf(a.Output, "Switched this repository to profile %q.\n", item.Name)
	}
	return nil
}

func (a App) status() error {
	state, err := a.Repo.State()
	if err != nil {
		return err
	}
	protocol := "ssh"
	if strings.HasPrefix(state.Origin, "https://") || strings.HasPrefix(state.Origin, "http://") {
		protocol = "https"
	}
	fmt.Fprintf(a.Output, "Repository: %s\nProfile:    %s\nProtocol:   %s\nGit user:   %s <%s>\nOrigin:     %s\n", state.Root, empty(state.Profile, "(none)"), protocol, empty(state.Name, "(unset)"), empty(state.Email, "(unset)"), state.Origin)
	return nil
}

func (a App) doctor() error {
	state, err := a.Repo.State()
	if err != nil {
		return err
	}
	settings, err := config.Load(a.Home)
	if err != nil {
		return err
	}
	fmt.Fprintf(a.Output, "Repository: %s\n", state.Root)
	if state.Profile == "" {
		return errors.New("no profile is bound to this repository")
	}
	item, found := settings.Find(state.Profile)
	if !found {
		return fmt.Errorf("bound profile %q does not exist", state.Profile)
	}
	if err := item.Validate(); err != nil {
		return err
	}
	remote, err := repo.ParseRemote(state.Origin)
	if err != nil {
		return err
	}
	if strings.HasPrefix(state.Origin, "https://") || strings.HasPrefix(state.Origin, "http://") {
		if err := item.ValidateHTTPS(); err != nil {
			return err
		}
		if err := gh.New().Check(remote.Host, item.GitHubUser); err != nil {
			return err
		}
		if state.Origin != remote.HTTPSURL() {
			return fmt.Errorf("origin does not use the expected HTTPS URL; expected %s", remote.HTTPSURL())
		}
		fmt.Fprintln(a.Output, "OK: profile, local Git identity, HTTPS origin, and GitHub CLI account are consistent.")
		return nil
	}
	if err := item.ValidateSSH(); err != nil {
		return err
	}
	expected := remote.SSHURL(item.HostAlias)
	if state.Origin != expected {
		return fmt.Errorf("origin does not use profile SSH alias; expected %s", expected)
	}
	hasHost, err := sshconfig.HasProfileHost(a.Home, item)
	if err != nil || !hasHost {
		return fmt.Errorf("SSH config does not contain a valid entry for %q", item.HostAlias)
	}
	fmt.Fprintln(a.Output, "OK: profile, local Git identity, origin, SSH key, and SSH config are consistent.")
	return nil
}

func (a App) restore(args []string) error {
	flags := flag.NewFlagSet("restore", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	yes := flags.Bool("yes", false, "")
	if err := flags.Parse(args); err != nil {
		return err
	}
	state, err := a.Repo.State()
	if err != nil {
		return err
	}
	if !*yes && !a.confirm(fmt.Sprintf("Restore Git settings for %s?", state.Root)) {
		return nil
	}
	if err := a.Repo.Restore(); err != nil {
		return err
	}
	fmt.Fprintln(a.Output, "Restored the previous repository settings.")
	return nil
}

func (a App) confirm(prompt string) bool {
	fmt.Fprintf(a.Output, "%s [y/N] ", prompt)
	line, err := bufio.NewReader(a.Input).ReadString('\n')
	return err == nil && strings.EqualFold(strings.TrimSpace(line), "y")
}

func (a App) usage() {
	fmt.Fprintln(a.Output, "Usage: gitid <account|use|status|doctor|restore>")
}

func empty(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
