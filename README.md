# Goth-example

## Пример авторизации с помощью Eve и Discord SSO

### Реквизиты

* Данные дискорд приложения https://discordapp.com/developers/applications/me

* Данные еве приложения https://developers.eveonline.com/applications

* Установленный Го https://golang.org/doc/install

### Теория

#### Го

* https://tour.golang.org/welcome/1

* https://rutracker.org/forum/viewtopic.php?t=5174202

#### Вебсервер

* https://www.quackit.com/web_servers/tutorial/how_web_servers_work.cfm

#### SSO

* на примере еве http://eveonline-third-party-documentation.readthedocs.io/en/latest/sso/

### Установка и использование

```bash
go get github.com/prusya/goth-example
cd $GOPATH/src/github.com/prusya/goth-example
go build .
./goth-example
```

Не забудь заполнить `config.json`

`EveClientId`, `EveClientSecret`, `EveCallback` в https://developers.eveonline.com/applications

`DiscordClientKey`, `DiscordClientSecret`, `DiscordCallback`, `DiscordToken` в https://discordapp.com/developers/applications/me

`DiscordGuildId` в клиенте дискорда пкм по гильдии -> `Copy ID`. Если нет этой строчки, в настройках `Appearance` включи `developer mode`.

`SessionStoreKey`, `GothicStoreKey` - рандомные строки.

`AllowedCharId`, `AllowedCorpId`, `AllowedAlliId` можно взять с зкб.

https://zkillboard.com/character/92532650/ eveCharId - это последние цифры
