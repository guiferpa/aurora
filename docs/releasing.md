# Publishing releases (maintainers)

Releases are built with [GoReleaser](https://goreleaser.com/) when you push a version tag (e.g. `v0.14.2`). The [Release workflow](https://github.com/guiferpa/aurora/actions/workflows/release.yml) runs automatically.

## Enabling Homebrew distribution

To allow users to install via `brew install guiferpa/tap/aurora`:

1. **Create a tap repository:** `guiferpa/homebrew-tap` on GitHub (empty or with a README).
2. **Create a [Personal Access Token](https://github.com/settings/tokens)** with `repo` scope.
3. **Add a repository secret** in the **aurora** repo:  
   **Settings → Secrets and variables → Actions → New repository secret**  
   Name: `HOMEBREW_TAP_TOKEN`  
   Value: the token from step 2.

After the next tagged release, the cask will be published to the tap.

## apt-get style install

Only `.deb` packages are published to the [Releases](https://github.com/guiferpa/aurora/releases) page. To offer `apt install aurora` you would need to host an APT repository (e.g. [Cloudsmith](https://cloudsmith.io/) or a custom repo) and point users to it.
