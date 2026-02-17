# Installing Aurora

## macOS (Homebrew)

```sh
brew tap guiferpa/tap
brew install guiferpa/tap/aurora
```

## Linux (Debian/Ubuntu)

Download the `.deb` from the [latest release](https://github.com/guiferpa/aurora/releases) and install:

```sh
sudo dpkg -i aurora_*_linux_*.deb
```

## Other platforms (pre-built binaries)

Download the [latest release](https://github.com/guiferpa/aurora/releases) and unpack the archive for your OS/arch.

## From source

You need [Go](https://go.dev/) installed.

```sh
go install -v github.com/guiferpa/aurora/cmd/aurora@HEAD
```

---

## macOS: “Apple could not verify …” (unverified developer)

If you see **“aurora” cannot be opened because the developer cannot be verified** or **Apple could not verify "aurora" is free of malware**, macOS Gatekeeper is blocking the unsigned binary. Use one of the options below.

### Option 1: Terminal (recommended)

In the folder where the `aurora` binary is (e.g. after unpacking a release):

```sh
xattr -cr aurora
./aurora
```

This removes the quarantine attribute so macOS allows the binary to run. You only need to do it once per binary.

To allow both CLI and LSP binary:

```sh
xattr -cr aurora aurorals
./aurora version
```

### Option 2: Finder (one-time approval)

1. **Right-click** (or Control-click) the `aurora` binary.
2. Choose **Open**.
3. In the dialog, click **Open** again.

macOS will remember your choice for that binary.

### Note for Homebrew users

The Homebrew cask applies the quarantine workaround automatically on install, so you typically won’t see this message when installing via `brew install guiferpa/tap/aurora`.
