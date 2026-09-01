# РосПанель — форк magwoo

Личный форк [AppsGanin/rospanel](https://github.com/AppsGanin/rospanel).

Этот форк собирается и выпускается GitHub Actions из этого репозитория. Установщик и встроенное самообновление используют **только** релизы `magwoo/rospanel`.

## Установка

```bash
curl -Ls https://raw.githubusercontent.com/magwoo/rospanel/main/install.sh | sudo bash
```

Установщик скачивает последний опубликованный Linux-бинарник под текущую архитектуру (`amd64` или `arm64`), проверяет его по `SHA256SUMS` из релиза, ставит systemd-сервис и выводит данные первого входа.

Чтобы установить конкретный релиз форка:

```bash
curl -Ls https://raw.githubusercontent.com/magwoo/rospanel/main/install.sh \
  | sudo ROSPANEL_VERSION=v2.11.0 bash
```

Готовые сборки:

- Releases: https://github.com/magwoo/rospanel/releases
- amd64: https://github.com/magwoo/rospanel/releases/latest/download/rospanel-linux-amd64
- arm64: https://github.com/magwoo/rospanel/releases/latest/download/rospanel-linux-arm64
- SHA256SUMS: https://github.com/magwoo/rospanel/releases/latest/download/SHA256SUMS

Тот же release channel используется встроенным обновлением панели и командами установки нод, которые генерируются в UI.

[English](README.md)
