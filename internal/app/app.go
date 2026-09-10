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

	"github.com/seanhooo/gitid/internal/config"
	"github.com/seanhooo/gitid/internal/gh"
	"github.com/seanhooo/gitid/internal/profile"
	"github.com/seanhooo/gitid/internal/repo"
	"github.com/seanhooo/gitid/internal/sshconfig"
	"github.com/seanhooo/gitid/internal/version"
)

type App struct {
	Home   string
	Input  io.Reader
	Output io.Writer
	Repo   repo.Repository
	Color  bool
}

func New() (App, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return App{}, err
	}
	return App{Home: home, Input: os.Stdin, Output: os.Stdout, Repo: repo.New(), Color: colorEnabled(os.Stdout)}, nil
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
	case "version", "--version", "-v":
		fmt.Fprintln(a.Output, version.String())
		return nil
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
	fmt.Fprintf(a.Output, "%s profile %q.\n", a.green("Added"), item.Name)
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
	fmt.Fprintf(a.Output, "%s profile %q.\n", a.green("Removed"), flags.Arg(0))
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
	fmt.Fprintf(a.Output, "%s %s\n", a.label("Repository:"), state.Root)
	fmt.Fprintf(a.Output, "%s %s <%s>\n", a.label("Profile:"), a.identity(item.GitName), a.strong(item.Email))
	fmt.Fprintf(a.Output, "%s %s\n", a.label("Protocol:"), *protocol)
	fmt.Fprintf(a.Output, "%s %s\n", a.label("Origin:"), newRemote)
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
		fmt.Fprintf(a.Output, "%s this repository to HTTPS profile %q.\n", a.green("Switched"), item.Name)
	} else {
		fmt.Fprintf(a.Output, "%s this repository to profile %q.\n", a.green("Switched"), item.Name)
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
	fmt.Fprintf(a.Output, "%s %s\n", a.label("Repository:"), state.Root)
	fmt.Fprintf(a.Output, "%s %s\n", a.label("Profile:"), a.identity(empty(state.Profile, "(none)")))
	fmt.Fprintf(a.Output, "%s %s\n", a.label("Protocol:"), protocol)
	fmt.Fprintf(a.Output, "%s %s <%s>\n", a.label("Git user:"), a.strong(empty(state.Name, "(unset)")), a.strong(empty(state.Email, "(unset)")))
	fmt.Fprintf(a.Output, "%s %s\n", a.label("Origin:"), state.Origin)
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
	fmt.Fprintf(a.Output, "%s %s\n", a.label("Repository:"), state.Root)
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
		fmt.Fprintf(a.Output, "%s profile, local Git identity, HTTPS origin, and GitHub CLI account are consistent.\n", a.ok())
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
	fmt.Fprintf(a.Output, "%s profile, local Git identity, origin, SSH key, and SSH config are consistent.\n", a.ok())
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
	fmt.Fprintf(a.Output, "%s the previous repository settings.\n", a.green("Restored"))
	return nil
}

func (a App) confirm(prompt string) bool {
	fmt.Fprintf(a.Output, "%s [y/N] ", prompt)
	line, err := bufio.NewReader(a.Input).ReadString('\n')
	return err == nil && strings.EqualFold(strings.TrimSpace(line), "y")
}

func (a App) usage() {
	out := a.Output
	fmt.Fprintf(out, "%s switches the Git identity (author name, email, and origin) used\n", a.prog())
	fmt.Fprintln(out, "by the current repository, without changing how you run git.")
	fmt.Fprintln(out)

	fmt.Fprintln(out, a.header("Usage:"))
	fmt.Fprintf(out, "  %s %s %s\n", a.prog(), a.dim("<command>"), a.dim("[flags]"))
	fmt.Fprintln(out)

	fmt.Fprintln(out, a.header("Commands:"))
	for _, c := range []struct {
		name, desc string
	}{
		{"account", "Manage named profiles"},
		{"use", "Apply a profile to the current repository"},
		{"status", "Show the current repository identity and origin"},
		{"doctor", "Verify the repository identity is consistent"},
		{"restore", "Roll back to the previous repository settings"},
		{"version", "Print version information"},
		{"help", "Show this help"},
	} {
		fmt.Fprintf(out, "  %s %s\n", a.green(padRight(c.name, 9)), c.desc)
	}
	fmt.Fprintln(out)

	fmt.Fprintln(out, a.header("Account subcommands:"))
	fmt.Fprintf(out, "  %s %s %s %s %s\n", a.prog(), a.green("account"), a.green("add"), a.dim("<name>"), a.dim("--git-name NAME --email EMAIL [flags]"))
	fmt.Fprintf(out, "  %s %s %s\n", a.prog(), a.green("account"), a.green("list"))
	fmt.Fprintf(out, "  %s %s %s %s\n", a.prog(), a.green("account"), a.green("remove"), a.dim("<name> [--yes]"))
	fmt.Fprintln(out)

	fmt.Fprintln(out, a.header("account add flags:"))
	fmt.Fprintf(out, "      %s %s\n", a.yellow("--git-name NAME"), "Git author name (required)")
	fmt.Fprintf(out, "      %s %s\n", a.yellow("--email EMAIL"), "Git author email (required)")
	fmt.Fprintf(out, "      %s %s\n", a.yellow("--github-user USER"), "GitHub username for HTTPS mode")
	fmt.Fprintf(out, "      %s %s\n", a.yellow("--key PATH"), "SSH private key path (~ is expanded)")
	fmt.Fprintf(out, "      %s %s\n", a.yellow("--host-alias ALIAS"), "SSH Host alias, e.g. github-work")
	fmt.Fprintf(out, "      %s %s\n", a.yellow("--host HOST"), "Remote hostname (default github.com)")
	fmt.Fprintln(out)

	fmt.Fprintln(out, a.header("use flags:"))
	fmt.Fprintf(out, "      %s %s\n", a.yellow("--protocol ssh|https"), "Protocol to use (default ssh)")
	fmt.Fprintf(out, "      %s %s\n", a.yellow("--yes"), "Apply without confirmation")
	fmt.Fprintln(out)

	fmt.Fprintln(out, a.header("restore flags:"))
	fmt.Fprintf(out, "      %s %s\n", a.yellow("--yes"), "Restore without confirmation")
	fmt.Fprintln(out)

	fmt.Fprintln(out, a.header("Examples:"))
	fmt.Fprintf(out, "  %s account add work --git-name \"Work Name\" --email work@example.com --key ~/.ssh/id_ed25519_work --host-alias github-work\n", a.prog())
	fmt.Fprintf(out, "  %s use work\n", a.prog())
	fmt.Fprintf(out, "  %s use work --protocol https\n", a.prog())
	fmt.Fprintf(out, "  %s status\n", a.prog())
	fmt.Fprintf(out, "  %s doctor\n", a.prog())
	fmt.Fprintf(out, "  %s restore\n", a.prog())
	fmt.Fprintln(out)

	fmt.Fprintln(out, "Profiles are stored in ~/.config/gitid/config.json. SSH Host aliases are")
	fmt.Fprintln(out, "managed in a marked block in ~/.ssh/config. HTTPS mode uses an already")
	fmt.Fprintln(out, "authenticated GitHub CLI (gh) account and never stores tokens.")
}

func empty(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
