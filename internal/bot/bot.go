// Package bot は Discord BOT 本体 (スラッシュコマンドとRSS通知) です。
package bot

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/KuronekoServer/RSS-BOT/internal/config"
	"github.com/KuronekoServer/RSS-BOT/internal/feed"
	"github.com/KuronekoServer/RSS-BOT/internal/store"
)

const embedColor = 0x0099FF

// Bot は Discord BOT です。
type Bot struct {
	cfg     *config.Config
	store   *store.Store
	fetcher *feed.Fetcher
	session *discordgo.Session

	readyOnce sync.Once
	ctx       context.Context
}

// New は Bot を作成します。
func New(cfg *config.Config, st *store.Store) (*Bot, error) {
	s, err := discordgo.New("Bot " + cfg.Discord.Token)
	if err != nil {
		return nil, err
	}
	s.Identify.Intents = discordgo.IntentsGuilds
	return &Bot{
		cfg:     cfg,
		store:   st,
		fetcher: feed.NewFetcher(cfg.RSS.UserAgent, cfg.RSS.RemoveQueryPrefixes),
		session: s,
	}, nil
}

// Run は ctx がキャンセルされるまで BOT を動かします。
func (b *Bot) Run(ctx context.Context) error {
	b.ctx = ctx
	b.session.AddHandler(b.onReady)
	b.session.AddHandler(b.onInteraction)
	if err := b.session.Open(); err != nil {
		return fmt.Errorf("Discordに接続できません: %w", err)
	}
	defer b.session.Close()
	<-ctx.Done()
	return nil
}

func (b *Bot) onReady(s *discordgo.Session, r *discordgo.Ready) {
	_ = s.UpdateStatusComplex(discordgo.UpdateStatusData{
		Status: string(discordgo.StatusOnline),
		Activities: []*discordgo.Activity{{
			Name: b.cfg.Discord.Name,
			Type: discordgo.ActivityTypeStreaming,
		}},
	})
	slog.Info("ログインしました", "user", r.User.String())

	// 再接続時にも Ready が来るため、初回のみ実行
	b.readyOnce.Do(func() {
		appID := b.cfg.Discord.ClientID
		if appID == "" {
			appID = r.User.ID
		}
		if err := b.registerCommands(appID); err != nil {
			slog.Error("コマンドの登録に失敗しました", "err", err)
		}
		go b.pollLoop(b.ctx)
	})
}

// registerCommands は設定に従ってスラッシュコマンドを登録します (既存のグローバルコマンドは置き換えられます)。
func (b *Bot) registerCommands(appID string) error {
	var perm *int64
	if *b.cfg.Commands.AdminOnly {
		p := int64(discordgo.PermissionAdministrator)
		perm = &p
	}
	guildOnly := &[]discordgo.InteractionContextType{discordgo.InteractionContextGuild}

	cmds := []*discordgo.ApplicationCommand{
		{
			Name:                     b.cfg.Commands.Register.Name,
			Description:              b.cfg.Commands.Register.Description,
			DefaultMemberPermissions: perm,
			Contexts:                 guildOnly,
		},
		{
			Name:                     b.cfg.Commands.Unregister.Name,
			Description:              b.cfg.Commands.Unregister.Description,
			DefaultMemberPermissions: perm,
			Contexts:                 guildOnly,
		},
	}
	created, err := b.session.ApplicationCommandBulkOverwrite(appID, "", cmds)
	if err != nil {
		return err
	}
	slog.Info("コマンドを登録しました", "count", len(created),
		"register", "/"+b.cfg.Commands.Register.Name, "unregister", "/"+b.cfg.Commands.Unregister.Name)
	return nil
}

func (b *Bot) onInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand || i.GuildID == "" || i.Member == nil {
		return
	}
	name := i.ApplicationCommandData().Name
	if name != b.cfg.Commands.Register.Name && name != b.cfg.Commands.Unregister.Name {
		return
	}
	if *b.cfg.Commands.AdminOnly && i.Member.Permissions&discordgo.PermissionAdministrator == 0 {
		b.replyError(s, i, "このコマンドは管理者のみ実行できます。")
		return
	}

	ctx, cancel := context.WithTimeout(b.ctx, 10*time.Second)
	defer cancel()

	var (
		msg string
		err error
		ok  bool
	)
	switch name {
	case b.cfg.Commands.Register.Name:
		ok, err = b.store.Register(ctx, i.GuildID, i.ChannelID)
		msg = b.cfg.Commands.Register.Message
		if !ok {
			msg = b.cfg.Commands.Register.AlreadyMessage
		}
	case b.cfg.Commands.Unregister.Name:
		ok, err = b.store.Unregister(ctx, i.GuildID)
		msg = b.cfg.Commands.Unregister.Message
		if !ok {
			msg = b.cfg.Commands.Unregister.AlreadyMessage
		}
	}
	if err != nil {
		slog.Error("コマンドの処理に失敗しました", "command", name, "guild", i.GuildID, "err", err)
		b.replyError(s, i, fmt.Sprintf("不明なエラーが発生しました。\n詳細:%s\n運営に問い合わせていただけると幸いです。", err))
		return
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{{Title: "お知らせ", Description: msg, Color: embedColor}},
		},
	})
}

func (b *Bot) replyError(s *discordgo.Session, i *discordgo.InteractionCreate, msg string) {
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:  discordgo.MessageFlagsEphemeral,
			Embeds: []*discordgo.MessageEmbed{{Title: "⚠️エラー", Description: msg, Color: 0xED4245}},
		},
	})
}

// pollLoop は設定された間隔でフィードを確認します。
func (b *Bot) pollLoop(ctx context.Context) {
	interval := time.Duration(b.cfg.RSS.IntervalMinutes) * time.Minute
	slog.Info("RSSの監視を開始します", "feeds", len(b.cfg.RSS.Links), "interval", interval)
	b.pollAll(ctx)
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			b.pollAll(ctx)
		}
	}
}

func (b *Bot) pollAll(ctx context.Context) {
	for _, link := range b.cfg.RSS.Links {
		if ctx.Err() != nil {
			return
		}
		if err := b.poll(ctx, link); err != nil {
			slog.Error("フィードの処理に失敗しました", "feed", link, "err", err)
		}
	}
}

func (b *Bot) poll(ctx context.Context, feedURL string) error {
	slog.Info("フィードを取得します", "feed", feedURL)
	f, err := b.fetcher.Fetch(ctx, feedURL)
	if err != nil {
		return err
	}

	known, err := b.store.FeedKnown(ctx, feedURL)
	if err != nil {
		return err
	}
	// 初めてのフィードは既存記事を既読にするだけで通知しない
	silent := !known && *b.cfg.RSS.SkipInitial

	for _, item := range f.Items {
		isNew, err := b.store.MarkSeen(ctx, item.Link)
		if err != nil {
			return err
		}
		if !isNew || silent {
			continue
		}
		slog.Info("新着記事", "feed", feedURL, "title", item.Title, "link", item.Link)
		b.broadcast(ctx, b.format(f, item))
	}

	if !known {
		if silent {
			slog.Info("新しいフィードのため既存記事は通知せず既読にしました", "feed", feedURL, "items", len(f.Items))
		}
		return b.store.MarkFeedKnown(ctx, feedURL)
	}
	return nil
}

func (b *Bot) format(f *feed.Feed, item feed.Item) string {
	title := item.Title
	if r := []rune(title); len(r) > 1500 {
		title = string(r[:1500]) + "…"
	}
	msg := strings.NewReplacer(
		"{title}", title,
		"{link}", item.Link,
		"{feed}", f.Title,
	).Replace(b.cfg.RSS.MessageFormat)
	if r := []rune(msg); len(r) > 2000 {
		msg = string(r[:2000])
	}
	return msg
}

func (b *Bot) broadcast(ctx context.Context, msg string) {
	targets, err := b.store.Targets(ctx)
	if err != nil {
		slog.Error("通知先の取得に失敗しました", "err", err)
		return
	}
	for _, t := range targets {
		if _, err := b.session.ChannelMessageSend(t.ChannelID, msg, discordgo.WithContext(ctx)); err != nil {
			slog.Error("メッセージの送信に失敗しました", "guild", t.ServerID, "channel", t.ChannelID, "err", err)
		}
	}
}

