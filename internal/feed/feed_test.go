package feed

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

const jvnRDF = `<?xml version="1.0" encoding="UTF-8"?>
<rdf:RDF xmlns="http://purl.org/rss/1.0/" xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#"
  xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:dcterms="http://purl.org/dc/terms/" xml:lang="ja">
 <channel rdf:about="https://jvn.jp/rss/jvn.rdf">
  <title>JVNRSS Feed - Update Entry</title>
  <link>https://jvn.jp/</link>
  <items><rdf:Seq>
   <rdf:li rdf:resource="https://jvn.jp/vu/JVNVU99905564/" />
   <rdf:li rdf:resource="https://jvn.jp/vu/JVNVU91023667/" />
  </rdf:Seq></items>
 </channel>
 <item rdf:about="https://jvn.jp/vu/JVNVU99905564/">
  <title>PHPにおける複数の脆弱性</title>
  <link>https://jvn.jp/vu/JVNVU99905564/</link>
  <dc:date>2026-10-22T10:30:00+09:00</dc:date>
 </item>
 <item rdf:about="https://jvn.jp/vu/JVNVU91023667/">
  <title>LIFF SDKにおけるクロスサイトスクリプティングの脆弱性</title>
  <link>https://jvn.jp/vu/JVNVU91023667/</link>
  <dc:date>2026-10-22T10:00:00+09:00</dc:date>
 </item>
</rdf:RDF>`

const wordpressRSS = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0"><channel><title>WP</title><link>https://example.com</link>
<item><title>新しい記事</title><link>https://example.com/new/?utm_source=rss&amp;utm_medium=rss&amp;p=2</link><pubDate>Wed, 21 Oct 2026 12:00:00 +0000</pubDate></item>
<item><title>古い記事</title><link>https://example.com/old/?utm_source=rss</link><pubDate>Tue, 20 Oct 2026 12:00:00 +0000</pubDate></item>
</channel></rss>`

const atom = `<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom"><title>Atom</title>
<entry><title>B</title><link href="https://example.org/b"/><id>b</id><updated>2026-10-21T00:00:00Z</updated></entry>
<entry><title>A</title><link href="https://example.org/a"/><id>a</id><updated>2026-10-20T00:00:00Z</updated></entry>
</feed>`

const jsonFeed = `{"version":"https://jsonfeed.org/version/1.1","title":"JSON",
"items":[{"id":"2","title":"Two","url":"https://example.net/2"},{"id":"1","title":"One","url":"https://example.net/1"}]}`

func TestFetchFormats(t *testing.T) {
	bodies := map[string]string{"/rdf": jvnRDF, "/rss": wordpressRSS, "/atom": atom, "/json": jsonFeed}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(bodies[r.URL.Path]))
	}))
	defer srv.Close()

	tests := []struct {
		path  string
		title string
		want  []Item
	}{
		{"/rdf", "JVNRSS Feed - Update Entry", []Item{
			{"LIFF SDKにおけるクロスサイトスクリプティングの脆弱性", "https://jvn.jp/vu/JVNVU91023667/"},
			{"PHPにおける複数の脆弱性", "https://jvn.jp/vu/JVNVU99905564/"},
		}},
		{"/rss", "WP", []Item{
			{"古い記事", "https://example.com/old/"},
			{"新しい記事", "https://example.com/new/?p=2"},
		}},
		{"/atom", "Atom", []Item{{"A", "https://example.org/a"}, {"B", "https://example.org/b"}}},
		{"/json", "JSON", []Item{{"One", "https://example.net/1"}, {"Two", "https://example.net/2"}}},
	}

	f := NewFetcher("test", []string{"utm_"})
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got, err := f.Fetch(context.Background(), srv.URL+tt.path)
			if err != nil {
				t.Fatal(err)
			}
			if got.Title != tt.title {
				t.Errorf("title = %q, want %q", got.Title, tt.title)
			}
			if len(got.Items) != len(tt.want) {
				t.Fatalf("items = %v, want %v", got.Items, tt.want)
			}
			for i := range tt.want {
				if got.Items[i] != tt.want[i] {
					t.Errorf("item[%d] = %v, want %v", i, got.Items[i], tt.want[i])
				}
			}
		})
	}
}

func TestFetchJVNLive(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}
	got, err := NewFetcher("RSS-BOT test", []string{"utm_"}).Fetch(context.Background(), "https://jvn.jp/rss/jvn.rdf")
	if err != nil {
		t.Skipf("network unavailable: %v", err)
	}
	if len(got.Items) == 0 {
		t.Fatal("no items")
	}
	t.Logf("%s: %d items, latest=%v", got.Title, len(got.Items), got.Items[len(got.Items)-1])
}

