# scdl

Soundcloud Downloader.

## Install (macOS/Linux)

The installer builds from source and places `scdl` in `~/.local/bin` by default.

```sh
curl -fsSL https://raw.githubusercontent.com/nathan-tomodachi/scdl/master/install.sh | sh
```

### Options

- `SCDL_INSTALL_DIR` to override the install directory.
- `SCDL_REF` to override the git ref (defaults to `master`).
- `SCDL_VERSION` to install a tagged release (e.g. `1.0.0` -> `v1.0.0`) and embed it in the binary.
- `SCDL_REPO_URL` to point at a different repo (useful for forks).
- `SCDL_REPO_NAME` to override the archive folder name (defaults to `scdl`).

Example:

```sh
SCDL_INSTALL_DIR="$HOME/bin" curl -fsSL https://raw.githubusercontent.com/nathan-tomodachi/scdl/master/install.sh | sh
```

Install a tagged release:

```sh
SCDL_VERSION="1.0.0" curl -fsSL https://raw.githubusercontent.com/nathan-tomodachi/scdl/master/install.sh | sh
```

### Requirements

- `curl`
- `tar`
- `go`

## Quick Start

### Initialize (recommended first run)

```sh
scdl init
```

`scdl init` launches a setup walkthrough that checks for dependencies (ffmpeg and yt-dlp or youtube-dl), offers to install missing packages, and writes a config file with your default output directory.

Dependencies for runtime:
- `ffmpeg`
- `yt-dlp` (or `youtube-dl`)

Build dependency:
- `go` (used by the installer to build the binary)

### TUI mode

```sh
scdl
```

Running `scdl` with no arguments opens the interactive TUI to enter a SoundCloud URL and choose output settings.

### CLI mode

```sh
scdl <soundcloud_url>
```

#### Flags

- `--config` path to a config file (default: `$HOME/.scdl.yaml`)
- `-o`, `--output` output directory (default: current directory or config value)
- `-f`, `--force` overwrite output file if it exists

### Config file

The default config file is created at `$HOME/.scdl.yaml`. It currently stores `output_dir` and is used as the default output directory when `-o/--output` is not provided.
