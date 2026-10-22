// Package config は config.json の読み込みと検証を行います。
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// Config は BOT 全体の設定です。
type Config struct {
	Discord  Discord  `json:"discord"`
	Commands Commands `json:"commands"`
	RSS      RSS      `json:"RSS"`
	Mongo    Mongo    `json:"mongo"`
}

// Discord は Discord 接続まわりの設定です。
type Discord struct {
	Token    string `json:"token"`
	ClientID string `json:"clientid"` // 省略時はログインしたBOTのIDを使用
	Name     string `json:"name"`     // ステータスに表示する名前
}

// Command は 1 つのスラッシュコマンドの設定です。
type Command struct {
	Name           string `json:"name"`            // コマンド名 (例: "rss-start")
	Description    string `json:"description"`     // コマンドの説明
	Message        string `json:"message"`         // 成功時のメッセージ
	AlreadyMessage string `json:"already_message"` // 既に登録済み/未登録の場合のメッセージ
}

// Commands は登録・削除コマンドの設定です。
type Commands struct {
	Register   Command `json:"register"`
	Unregister Command `json:"unregister"`
	AdminOnly  *bool   `json:"admin_only"` // 管理者のみ実行可能にするか (既定: true)
}

// RSS はフィード取得の設定です。
type RSS struct {
	Links               []string `json:"links"`
	IntervalMinutes     int      `json:"interval_minutes"`      // 取得間隔(分) 既定: 5
	SkipInitial         *bool    `json:"skip_initial"`          // 初めて取得するフィードの既存記事を通知しない (既定: true)
	MessageFormat       string   `json:"message_format"`        // 通知メッセージの書式 ({title} {link} {feed})
	RemoveQueryPrefixes []string `json:"remove_query_prefixes"` // URLから削除するクエリパラメータの接頭辞 (既定: ["utm_"])
	UserAgent           string   `json:"user_agent"`
}

// Mongo は MongoDB の設定です。
type Mongo struct {
	Link     string `json:"link"`
	Database string `json:"database"` // 省略時は接続URIのDB名、それも無ければ "test" (mongoose互換)
}

var commandNameRe = regexp.MustCompile(`^[-_\p{L}\p{N}]{1,32}$`)

// Load は指定パスの設定ファイルを読み込み、既定値の補完と検証を行います。
func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("設定ファイルを読み込めません: %w", err)
	}
	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("設定ファイルの形式が不正です: %w", err)
	}
	c.applyDefaults()
	if err := c.validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

func (c *Config) applyDefaults() {
	if c.Discord.Name == "" {
		c.Discord.Name = "RSS"
	}

	r := &c.Commands.Register
	if r.Name == "" {
		r.Name = "rss-start"
	}
	if r.Description == "" {
		r.Description = "このチャンネルでRSS通知を開始します"
	}
	if r.Message == "" {
		r.Message = "保存しました。"
	}
	if r.AlreadyMessage == "" {
		r.AlreadyMessage = "登録済み"
	}

	u := &c.Commands.Unregister
	if u.Name == "" {
		u.Name = "rss-stop"
	}
	if u.Description == "" {
		u.Description = "RSS通知を停止します"
	}
	if u.Message == "" {
		u.Message = "削除しました。"
	}
	if u.AlreadyMessage == "" {
		u.AlreadyMessage = "未登録"
	}

	if c.Commands.AdminOnly == nil {
		t := true
		c.Commands.AdminOnly = &t
	}

	if c.RSS.IntervalMinutes <= 0 {
		c.RSS.IntervalMinutes = 5
	}
	if c.RSS.SkipInitial == nil {
		t := true
		c.RSS.SkipInitial = &t
	}
	if c.RSS.MessageFormat == "" {
		c.RSS.MessageFormat = "{title}\n{link}"
	}
	if c.RSS.RemoveQueryPrefixes == nil {
		c.RSS.RemoveQueryPrefixes = []string{"utm_"}
	}
	if c.RSS.UserAgent == "" {
		c.RSS.UserAgent = "RSS-BOT (+https://github.com/KuronekoServer/RSS-BOT)"
	}

	if c.Mongo.Database == "" {
		c.Mongo.Database = databaseFromURI(c.Mongo.Link)
	}
}

func (c *Config) validate() error {
	var errs []error
	if c.Discord.Token == "" {
		errs = append(errs, errors.New("discord.token が設定されていません"))
	}
	if c.Mongo.Link == "" {
		errs = append(errs, errors.New("mongo.link が設定されていません"))
	}
	if len(c.RSS.Links) == 0 {
		errs = append(errs, errors.New("RSS.links が空です"))
	}
	for key, cmd := range map[string]Command{"register": c.Commands.Register, "unregister": c.Commands.Unregister} {
		if !commandNameRe.MatchString(cmd.Name) || strings.ToLower(cmd.Name) != cmd.Name {
			errs = append(errs, fmt.Errorf("commands.%s.name %q はDiscordのコマンド名として使えません (小文字・数字・-・_ で1〜32文字)", key, cmd.Name))
		}
		if n := len([]rune(cmd.Description)); n < 1 || n > 100 {
			errs = append(errs, fmt.Errorf("commands.%s.description は1〜100文字にしてください", key))
		}
	}
	if c.Commands.Register.Name == c.Commands.Unregister.Name {
		errs = append(errs, errors.New("commands.register.name と commands.unregister.name が同じです"))
	}
	return errors.Join(errs...)
}

// databaseFromURI は mongodb://user:pass@host/db?opt の db 部分を取り出します。
func databaseFromURI(uri string) string {
	s := uri
	if i := strings.Index(s, "://"); i >= 0 {
		s = s[i+3:]
	}
	if i := strings.LastIndex(s, "@"); i >= 0 {
		s = s[i+1:]
	}
	if i := strings.Index(s, "/"); i >= 0 {
		s = s[i+1:]
		if j := strings.IndexAny(s, "?#"); j >= 0 {
			s = s[:j]
		}
		if s != "" {
			return s
		}
	}
	return "test"
}

