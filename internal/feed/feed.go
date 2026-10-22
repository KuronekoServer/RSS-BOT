// Package feed は各種フィード (RSS 0.9x/1.0(RDF)/2.0, Atom, JSON Feed) の取得を行います。
package feed

import (
	"context"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
)

// Item は通知対象の記事です。
type Item struct {
	Title string
	Link  string
}

// Feed は取得結果です。
type Feed struct {
	Title string
	Items []Item // 古い順
}

// Fetcher はフィードを取得します。
type Fetcher struct {
	parser              *gofeed.Parser
	removeQueryPrefixes []string
}

// NewFetcher は Fetcher を作成します。
func NewFetcher(userAgent string, removeQueryPrefixes []string) *Fetcher {
	p := gofeed.NewParser()
	p.UserAgent = userAgent
	return &Fetcher{parser: p, removeQueryPrefixes: removeQueryPrefixes}
}

// Fetch は URL のフィードを取得し、記事を古い順に返します。
// フィード形式は自動判別されます。
func (f *Fetcher) Fetch(ctx context.Context, feedURL string) (*Feed, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	parsed, err := f.parser.ParseURLWithContext(feedURL, ctx)
	if err != nil {
		return nil, err
	}

	items := slices.Clone(parsed.Items)
	sortOldestFirst(items)

	out := &Feed{Title: strings.TrimSpace(parsed.Title)}
	for _, it := range items {
		link := itemLink(it)
		if link == "" {
			continue
		}
		title := strings.TrimSpace(it.Title)
		if title == "" {
			title = link
		}
		out.Items = append(out.Items, Item{Title: title, Link: CleanLink(link, f.removeQueryPrefixes)})
	}
	return out, nil
}

// itemLink は記事のURLを返します。link が無い場合は GUID / RDF の about がURLならそれを使います。
func itemLink(it *gofeed.Item) string {
	if l := strings.TrimSpace(it.Link); l != "" {
		return l
	}
	for _, l := range it.Links {
		if l = strings.TrimSpace(l); l != "" {
			return l
		}
	}
	if g := strings.TrimSpace(it.GUID); strings.HasPrefix(g, "http://") || strings.HasPrefix(g, "https://") {
		return g
	}
	return ""
}

func itemTime(it *gofeed.Item) *time.Time {
	if it.PublishedParsed != nil {
		return it.PublishedParsed
	}
	return it.UpdatedParsed
}

// sortOldestFirst は全記事に日時があれば日時順、無ければフィード順を反転します
// (多くのフィードは新しい順に並んでいるため)。
func sortOldestFirst(items []*gofeed.Item) {
	for _, it := range items {
		if itemTime(it) == nil {
			slices.Reverse(items)
			return
		}
	}
	slices.SortStableFunc(items, func(a, b *gofeed.Item) int {
		return itemTime(a).Compare(*itemTime(b))
	})
}

// CleanLink は指定した接頭辞 (例: "utm_") で始まるクエリパラメータを削除します。
// 残りのパラメータの順序は維持します。
func CleanLink(raw string, prefixes []string) string {
	u, err := url.Parse(raw)
	if err != nil || u.RawQuery == "" || len(prefixes) == 0 {
		return raw
	}
	var kept []string
	for _, kv := range strings.Split(u.RawQuery, "&") {
		if kv == "" {
			continue
		}
		key := kv
		if i := strings.Index(kv, "="); i >= 0 {
			key = kv[:i]
		}
		if k, err := url.QueryUnescape(key); err == nil {
			key = k
		}
		if !slices.ContainsFunc(prefixes, func(p string) bool { return strings.HasPrefix(key, p) }) {
			kept = append(kept, kv)
		}
	}
	u.RawQuery = strings.Join(kept, "&")
	return u.String()
}

