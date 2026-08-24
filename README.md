# gitid

`gitid` switches the identity used by the current Git repository without changing the Git commands you normally run. After `gitid use work`, continue with `git push`, `git pull`, and `git fetch` as usual.

## Features

- Named profiles with Git author identity, SSH metadata, and an optional GitHub CLI username
- SSH `Host` aliases managed in a marked block in `~/.ssh/config`
- Repository-local Git identity and `origin` rewrite for SSH or HTTPS
- HTTPS mode that selects an already-authenticated GitHub CLI (`gh`) account without storing tokens
- `status`, consistency diagnostics, and repository rollback

## Build

```sh
go build -o gitid ./cmd/gitid
```

## Add profiles

Create a distinct SSH key for each account before adding it. `gitid` validates the key path but never reads private-key contents.

```sh
./gitid account add work \
  --git-name "Work Name" \
  --email work@example.com \
  --key ~/.ssh/id_ed25519_work \
  --host-alias github-work

./gitid account add personal \
  --git-name "Personal Name" \
  --email personal@example.com \
  --key ~/.ssh/id_ed25519_personal \
  --host-alias github-personal
```

The profiles are stored at `~/.config/gitid/config.json` with `0600` permissions. The CLI creates a managed SSH configuration block such as:

```sshconfig
# >>> gitid managed block >>>
Host github-work
  HostName github.com
  User git
  IdentityFile /Users/you/.ssh/id_ed25519_work
  IdentitiesOnly yes
# <<< gitid managed block <<<
```

All unrelated content in `~/.ssh/config` is preserved. `gitid` refuses to use an alias already defined outside its managed block.

## Switch a repository

Run this inside an existing repository with an `origin` remote:

```sh
./gitid use work
```

The command previews the repository, commit identity, and rewritten origin before asking for confirmation. Use `--yes` for scripts:

```sh
./gitid use work --yes
```

For a GitHub repository, its origin becomes:

```text
git@github-work:owner/repository.git
```

The command writes the selected identity as repository-local `user.name`, `user.email`, and `gitid.profile`. It saves the previous local settings and origin before the first mutation.

## HTTPS with GitHub CLI

For HTTPS, Gitid delegates credentials to GitHub CLI. Authenticate each account before configuring or switching a profile:

```sh
gh auth login --hostname github.com
# Repeat and select the other GitHub account.
```

Add the GitHub username to the profile. SSH fields are optional for an HTTPS-only profile:

```sh
./gitid account add personal \
  --git-name "Personal Name" \
  --email personal@example.com \
  --github-user personal-github-login
```

In a repository, select HTTPS mode:

```sh
./gitid use personal --protocol https
```

Gitid verifies that `personal-github-login` is already authenticated through `gh`, switches `gh`'s active account, and rewrites the origin to `https://github.com/owner/repository.git`. Gitid never stores, reads, or displays a token. If the account is not available, authenticate it manually with `gh auth login`.

SSH remains the default:

```sh
./gitid use work --protocol ssh
```

Because the active GitHub CLI account is global, `gitid restore` restores the repository's local identity and origin but does not change the active `gh` account.


```sh
./gitid status
./gitid doctor
./gitid restore
```

`doctor` validates the selected profile against the active protocol. SSH checks key existence and the managed SSH entry; HTTPS checks that the configured GitHub account is already authenticated through `gh`. `restore` returns the current repository to the settings saved before the first `use`.

## Limitations

- Supported origins are single-level `owner/repository` SSH or HTTP(S) URLs.
- SSH remains the default protocol; HTTPS currently supports GitHub accounts authenticated through `gh`.
- Gitid does not run `git push`, call `gh auth login`, or manage access tokens itself.
