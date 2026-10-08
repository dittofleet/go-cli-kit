# go-cli-kit

What every dittofleet Go CLI does the same way: where it keeps its files, how it is installed and updates itself, the daily hint that a newer release is out, and how releases get built.

Each CLI describes itself once:

```go
app := clikit.App{Name: "navi", Version: version}
```

`Name` is both the binary and its repo under dittofleet. The app is handed to the packages:

| Package | What it does |
|---|---|
| `xdg` | `ConfigDir(name)` and `DataDir(name)`, ignoring relative `XDG_*` values |
| `updatecheck` | `MaybeCheck(app, command)` prints a hint at most once a day when a newer release is out. Skipped after `update`, `postinstall` and `uninstall`, for dev builds, in CI, when `<APP>_NO_UPDATE_CHECK` is set, and when stderr is not a terminal |
| `postinstall` | `Run(app, setup)` runs a CLI's first-time setup once per machine, for the [shared install script](https://github.com/dittofleet/.github/blob/main/install.sh). After that it says the CLI is already set up |
| `selfupdate` | `Run(app)` installs the latest release over the running binary, and reports whether it did |
| `uninstall` | `Run(app, yes, plan)` lists what an `uninstall` command removes, asks first (or takes `--yes`), and removes the plan's items, the kit's own files and then the binary, saying what was removed if a step fails |
| `release` | Asset URLs, and the latest-tag fetch, with GitHub's rate limit explained |

`clikit.Executable()` resolves the running binary through symlinks, for update and uninstall commands.

## Releases

A CLI's release workflow calls the shared one:

```yaml
name: Release

on:
  push:
    tags: ["v*"]

permissions:
  contents: write

jobs:
  release:
    uses: dittofleet/go-cli-kit/.github/workflows/go-release.yml@main
    # with:
    #   targets: darwin-arm64 darwin-x64 # defaults to macOS and Linux, arm64 and x64
```

It checks the tag with [check-release-tag](https://github.com/dittofleet/.github/tree/main/actions/check-release-tag) (`vX.Y.Z`, on main), runs the tests, builds `<repo>-<os>-<arch>` for each target with the tag in `main.version`, and publishes them as a release.

## Starting a new CLI

1. Describe it with a `clikit.App`, call `updatecheck.MaybeCheck` after each command, `selfupdate.Run` for `update`, `uninstall.Run` for `uninstall` and `postinstall.Run` for a hidden `postinstall`.
2. Install it with the shared script:

```sh
curl -fsSL https://raw.githubusercontent.com/dittofleet/.github/main/install.sh | sh -s <name>
```

3. Add the release workflow from the "Go CLI release" template (Actions → New workflow), or copy the snippet above.
4. Add `.github/dependabot.yml`, so new kit versions open a PR the next day, skipping Dependabot's default 3-day wait:

```yaml
version: 2
updates:
  - package-ecosystem: gomod
    directory: /
    schedule:
      interval: daily
    allow:
      - dependency-name: github.com/dittofleet/go-cli-kit
    cooldown:
      exclude:
        - github.com/dittofleet/go-cli-kit
```

## Working on the kit and a CLI together

Point `GOWORK` at a workspace outside the repos, so other checkouts aren't affected. Replacing the kit, rather than listing it under `use`, also covers a CLI that requires a kit version that isn't tagged yet:

```sh
# From the folder holding both checkouts
cat > /tmp/kit.work <<EOF
go 1.26.5
use $PWD/navi
replace github.com/dittofleet/go-cli-kit => $PWD/go-cli-kit
EOF
cd navi && GOWORK=/tmp/kit.work go test ./...
```
