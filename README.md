# RosPanel — magwoo fork

Personal fork of [AppsGanin/rospanel](https://github.com/AppsGanin/rospanel).

This fork is built and released by GitHub Actions from this repository. The installer and the built-in self-updater use **only** `magwoo/rospanel` releases.

## Install

```bash
curl -Ls https://raw.githubusercontent.com/magwoo/rospanel/main/install.sh | sudo bash
```

The installer downloads the latest published Linux binary for the current architecture (`amd64` or `arm64`), verifies it against the release `SHA256SUMS`, installs the systemd service and prints first-run credentials.

To pin a specific fork release:

```bash
curl -Ls https://raw.githubusercontent.com/magwoo/rospanel/main/install.sh \
  | sudo ROSPANEL_VERSION=v2.11.0 bash
```

Published builds are available here:

- Releases: https://github.com/magwoo/rospanel/releases
- amd64: https://github.com/magwoo/rospanel/releases/latest/download/rospanel-linux-amd64
- arm64: https://github.com/magwoo/rospanel/releases/latest/download/rospanel-linux-arm64
- checksums: https://github.com/magwoo/rospanel/releases/latest/download/SHA256SUMS

The same release channel is used by panel self-updates and by node installation commands generated in the UI.

[Русский](README-RU.md)
