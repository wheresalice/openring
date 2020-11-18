package main

import (
	"bufio"
	"html"
	"html/template"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"git.sr.ht/~sircmpwn/getopt"

	"github.com/SlyMarbo/rss"
	"github.com/mattn/go-runewidth"
	"github.com/microcosm-cc/bluemonday"
)

type urlSlice []*url.URL

func (us *urlSlice) String() string {
	var str []string
	for _, u := range *us {
		str = append(str, u.String())
	}
	return strings.Join(str, ", ")
}

func (us *urlSlice) Set(val string) error {
	u, err := url.Parse(val)
	if err != nil {
		return err
	}
	*us = append(*us, u)
	return nil
}

type Article struct {
	Date        time.Time
	Link        string
	SourceLink  string
	SourceTitle string
	Summary     template.HTML
	Title       string
}

func main() {
	var (
		narticles  = getopt.Int("n", 3, "article count")
		perSource  = getopt.Int("p", 1, "articles to take from each source")
		summaryLen = getopt.Int("l", 256, "length of summaries")
		urlsFile   = getopt.String("S", "", "file with URLs of sources")
		sources    []*url.URL
	)
	getopt.Var((*urlSlice)(&sources), "s", "list of sources")

	getopt.Usage = func() {
		log.Fatalf("Usage: %s [-s https://source.rss...] < in.html > out.html",
			os.Args[0])
	}

	err := getopt.Parse()
	if err != nil {
		panic(err)
	}

	if *urlsFile != "" {
		file, err := os.Open(*urlsFile)
		if err != nil {
			panic(err)
		}
		sc := bufio.NewScanner(file)
		for sc.Scan() {
			(*urlSlice)(&sources).Set(sc.Text())
		}
		file.Close()
	}

	input, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}

	tmpl, err := template.
		New("template").
		Funcs(map[string]interface{}{
			"date": func(t time.Time) string {
				return t.Format("January 2, 2006")
			},
			"datef": func(fmt string, t time.Time) string {
				return t.Format(fmt)
			},
		}).
		Parse(string(input))
	if err != nil {
		panic(err)
	}

	log.Println("Fetching feeds...")
	var feeds []*rss.Feed
	for _, source := range sources {
		feed, err := rss.Fetch(source.String())
		if err != nil {
			log.Printf("Error fetching %s: %s", source.String(), err.Error())
			continue
		}
		feeds = append(feeds, feed)
		log.Printf("Fetched %s", feed.Title)
	}
	if len(feeds) == 0 {
		log.Fatal("Expected at least one feed to successfully fetch")
	}

	policy := bluemonday.StrictPolicy()

	var articles []*Article
	for _, feed := range feeds {
		if len(feed.Items) == 0 {
			log.Printf("Warning: feed %s has no items", feed.Title)
			continue
		}
		items := feed.Items
		if len(items) > *perSource {
			items = items[:*perSource]
		}
		base, err := url.Parse(feed.UpdateURL)
		if err != nil {
			log.Fatal("failed parsing update URL of the feed")
		}
		feedLink, _ := url.Parse(feed.Link)
		if err != nil {
			log.Fatal("failed parsing canonical feed URL of the feed")
		}
		for _, item := range items {
			raw_summary := item.Summary
			if len(raw_summary) == 0 {
				raw_summary = html.UnescapeString(item.Content)
			}
			summary := runewidth.Truncate(
				policy.Sanitize(raw_summary), *summaryLen, "…")

			itemLink, _ := url.Parse(item.Link)
			if err != nil {
				log.Fatal("failed parsing article URL of the feed item")
			}

			articles = append(articles, &Article{
				Date:        item.Date,
				SourceLink:  base.ResolveReference(feedLink).String(),
				SourceTitle: feed.Title,
				Summary:     template.HTML(summary),
				Title:       item.Title,
				Link:        base.ResolveReference(itemLink).String(),
			})
		}
	}
	sort.Slice(articles, func(i, j int) bool {
		return articles[i].Date.After(articles[j].Date)
	})
	if len(articles) < *narticles {
		*narticles = len(articles)
	}
	articles = articles[:*narticles]
	err = tmpl.Execute(os.Stdout, struct {
		Articles []*Article
	}{
		Articles: articles,
	})
	if err != nil {
		panic(err)
	}
}
