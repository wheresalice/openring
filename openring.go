package main

import (
	"html/template"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/SlyMarbo/rss"
	"github.com/mattn/go-runewidth"
	"github.com/microcosm-cc/bluemonday"
	"git.sr.ht/~sircmpwn/getopt"
)

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
		narticles  int        = 3
		summaryLen int        = 256
		sources    []*url.URL
	)

	opts, optind, err := getopt.Getopts(os.Args[1:], "l:n:s:")
	if err != nil {
		panic(err)
	}
	for _, opt := range opts {
		switch opt.Option {
		case 'l':
			summaryLen, err = strconv.Atoi(opt.Value)
			if err != nil {
				panic(err)
			}
		case 'n':
			narticles, err = strconv.Atoi(opt.Value)
			if err != nil {
				panic(err)
			}
		case 's':
			u, err := url.Parse(opt.Value)
			if err != nil {
				panic(err)
			}
			sources = append(sources, u)
		}
	}

	if len(os.Args[optind+1:]) != 0 {
		log.Fatalf(
			"Usage: %s [-s https://source.rss...] < in.html > out.html",
			os.Args[0])
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
		item := feed.Items[0]
		summary := runewidth.Truncate(
			policy.Sanitize(item.Summary), summaryLen, "…")
		articles = append(articles, &Article{
			Date:        item.Date,
			SourceLink:  feed.Link,
			SourceTitle: feed.Title,
			Summary:     template.HTML(summary),
			Title:       item.Title,
			Link:        item.Link,
		})
	}
	sort.Slice(articles, func(i, j int) bool {
		return articles[i].Date.After(articles[j].Date)
	})
	articles = articles[:narticles]
	err = tmpl.Execute(os.Stdout, struct{
		Articles []*Article
	}{
		Articles: articles,
	})
	if err != nil {
		panic(err)
	}
}
