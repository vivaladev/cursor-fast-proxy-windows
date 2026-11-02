# Cursor fast proxy для Windows

Окно для Windows 10 и 11. После «Вжух и прокси» Cursor ходит в своё облако через локальный sing-box на `127.0.0.1:18081` и VPS. Системный прокси Windows, TUN и таблица маршрутов не меняются.

В программе нет чужих IP, паролей и ключей. Их вводят в настройках.

## Кнопки

- **Вжух и прокси** — скачивает sing-box 1.14.0, если его ещё нет, поднимает sing-box на VPS, локальный прокси и настройки Cursor.
- **Удалить настройки прокси** — «Да» останавливает локальную задачу и убирает прокси из Cursor. «Да, и на vps» дополнительно удаляет сервис `sing-box-cursor` и `/etc/sing-box-cursor`. AmneziaWG, SSH и бинарь sing-box остаются.
- **Настройки** — поля `ip`, `user`, `password`. «Сохранить» пишет их в `%USERPROFILE%\.cursor-fast-proxy\settings.env`.
- **Логи** — журнал команд. Значения ip, user и password заменены именами переменных.

Другой VPS: удалить локальный прокси, вписать новые ip/user/password, снова нажать «Вжух и прокси».

После включения или удаления нужен полный перезапуск Cursor, не Reload Window.

Нужна 64-битная Windows 10 или 11 (amd64 или arm64).

## Сборка

Нужен Go 1.23+. С macOS или Linux:

```bash
go test ./...
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "-H windowsgui" -o dist/CursorFastProxy.exe ./cmd/cursor-fast-proxy
```

Для Windows on ARM замени `GOARCH=amd64` на `arm64`.

Версия берётся из файла `VERSION`.
