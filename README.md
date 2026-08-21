# gitid

`gitid` switches the SSH identity used by the current Git repository without changing the Git commands you normally run. After `gitid use work`, continue with `git push`, `git pull`, and `git fetch` as usual.

## Phase 1 scope

- Named profiles with Git author identity and SSH key paths
- SSH `Host` aliases managed in a marked block in `~/.ssh/config`
- Repository-local Git identity and `origin` rewrite
- `status`, consistency diagnostics, and repository rollback

HTTPS credential switching is intentionally deferred. An HTTPS origin can be converted to the selected profile's SSH alias by `gitid use`, but this version does not store or switch HTTPS credentials.

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

## Inspect and recover

```sh
./gitid status
./gitid doctor
./gitid restore
```

`doctor` is offline-only in phase 1: it checks profile data, key existence, origin format, and the managed SSH entry; it does not contact GitHub. `restore` returns the current repository to the settings saved before the most recent `use`.

## Limitations

- Supported origins are single-level `owner/repository` SSH or HTTP(S) URLs.
- SSH is the authentication mechanism for phase 1.
- This release does not switch HTTPS credentials, authenticate to Git hosting providers, or run `git push` itself.
