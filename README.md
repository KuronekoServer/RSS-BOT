# RSS-BOT
## 制作
- [@MaruyamaKoushirou](<https://github.com/MaruyamaKoushirou>)
- [@kuroneko6423](<https://github.com/kuroneko6423>)

---
## 本レポジトリについて
本DiscordBOTはWordPressの記事が投稿されて、自動的にDiscordのチャンネルに通知される物です。
- DBではMongoDBを採用しています。
- 通知を有効・無効のコマンドだけのシンプルなものになっています。
- 複数のWordPressのサイトに対応しています。(1つのみでも可)
- 5分ごとにRSSを取得し、記事情報を確認しています。

例: https://support.krnk.org/feed/

※WordPress以外でのRSSの動作はテストしていません。

#### Discordの送信するメッセージ内容
- 記事のタイトル
- 記事のURL

プレビュー

<img width="446" height="328" alt="image" src="https://github.com/user-attachments/assets/607fca53-e387-410b-96a2-57262ab717b7" />
