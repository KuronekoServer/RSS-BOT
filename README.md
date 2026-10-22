# RSS-BOT
## 制作
- [@MaruyamaKoushirou](<https://github.com/MaruyamaKoushirou>)
- [@kuroneko6423](<https://github.com/kuroneko6423>)

---
## 本レポジトリについて
本DiscordBOTはRSSフィードに記事が投稿されると、自動的にDiscordのチャンネルに通知する物です。
- Go製です。(discordgo / gofeed / MongoDB Go Driver v2)
- DBではMongoDBを採用しています。(旧Node.js版のデータをそのまま引き継げます)
- 通知を有効・無効のコマンドだけのシンプルなものになっています。コマンド名は設定で変更できます。
- 複数のフィードに対応しています。(1つのみでも可)
- 設定した間隔(既定5分)ごとにRSSを取得し、記事情報を確認しています。

### 対応フィード形式
形式は自動判別されます。
- RSS 2.0 (WordPress など) 例: https://support.krnk.org/feed/
- RSS 1.0 / RDF 例: https://jvn.jp/rss/jvn.rdf
- RSS 0.9x
- Atom
- JSON Feed

#### Discordの送信するメッセージ内容
- 記事のタイトル
- 記事のURL

(`message_format` で変更できます)

プレビュー

<img width="446" height="328" alt="image" src="https://github.com/user-attachments/assets/607fca53-e387-410b-96a2-57262ab717b7" />

---
## 使い方
`config_sample.json` を `config.json` にコピーして編集し、起動します。

```sh
go run .                      # 開発時
go build -o rss-bot . && ./rss-bot -config config.json
```

Docker:
```sh
docker build -t rss-bot .
docker run -d -v ./config.json:/app/config.json rss-bot
```

## 設定 (config.json)
| キー | 説明 | 既定値 |
| --- | --- | --- |
| `discord.token` | BOTトークン | (必須) |
| `discord.clientid` | アプリケーションID | ログインしたBOTのID |
| `discord.name` | ステータスに表示する名前 | `RSS` |
| `commands.admin_only` | 管理者のみコマンドを実行可能にする | `true` |
| `commands.register.name` | 登録コマンド名 | `rss-start` |
| `commands.register.description` | 登録コマンドの説明 | |
| `commands.register.message` | 登録成功時のメッセージ | `保存しました。` |
| `commands.register.already_message` | 既に登録済みの時のメッセージ | `登録済み` |
| `commands.unregister.name` | 削除コマンド名 | `rss-stop` |
| `commands.unregister.description` | 削除コマンドの説明 | |
| `commands.unregister.message` | 削除成功時のメッセージ | `削除しました。` |
| `commands.unregister.already_message` | 未登録の時のメッセージ | `未登録` |
| `RSS.links` | 監視するフィードURL | (必須) |
| `RSS.interval_minutes` | 取得間隔(分) | `5` |
| `RSS.skip_initial` | 初めて取得するフィードの既存記事は通知せず既読にする | `true` |
| `RSS.message_format` | 通知メッセージ (`{title}` `{link}` `{feed}`) | `{title}\n{link}` |
| `RSS.remove_query_prefixes` | URLから削除するクエリパラメータの接頭辞 | `["utm_"]` |
| `mongo.link` | MongoDB接続URI | (必須) |
| `mongo.database` | DB名 | URI内のDB名、無ければ `test` (旧版と同じ) |

> コマンド名はDiscordの制限により、小文字・数字・`-`・`_` の1〜32文字です。
> 起動時にグローバルコマンドを上書き登録するため、コマンド名を変更すると古いコマンドは自動で消えます。
