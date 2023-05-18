// License-Id: GPL-3.0-only
// Copyright: 2019 Drew DeVault <sir@cmpwn.com>
// Copyright: 2019 Haelwenn (lanodan) Monnier <contact@hacktivis.me>
// Copyright: 2019 Jeff Kaufman <jeff.t.kaufman@gmail.com>
// Copyright: 2019 Nate Dobbins <nated@posteo.net>
// Copyright: 2019 Noah Loomans <noah@noahloomans.com>
// Copyright: 2019 Philip K <philip@warpmail.net>
// Copyright: 2019 Simon Ser <contact@emersion.fr>
// Copyright: 2020 Drew DeVault <sir@cmpwn.com>
// Copyright: 2020 skuzzymiglet <skuzzymiglet@gmail.com>
// Copyright: 2021 Gianluca Arbezzano <ciao@gianarb.it>
// Copyright: 2021 sourque <contact@sourque.com>
// Copyright: 2023 wheresalice
package main

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"html"
	"html/template"
	"io"
	"log"
	"net/url"
	"os"
	"sort"
	"time"

	"github.com/SlyMarbo/rss"
	"github.com/mattn/go-runewidth"
	"github.com/microcosm-cc/bluemonday"
)

type Article struct {
	Date        time.Time
	Link        string
	SourceLink  string
	SourceTitle string
	Summary     template.HTML
	Title       string
}

type Site struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
	RSS  string `yaml:"rss"`
}

func main() {
	var (
		articlesCount   = 3
		articlesPerSite = 1
		summaryLength   = 256
	)

	if len(os.Args) != 2 {
		log.Fatal("Usage: openring site.yml < in.html")
	}

	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	var sites []Site
	err = yaml.Unmarshal(data, &sites)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%+v\n", sites)

	input, err := io.ReadAll(os.Stdin)
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
	for _, site := range sites {
		feed, err := rss.Fetch(site.RSS)
		if err != nil {
			log.Printf("Error fetching %s: %s", site.RSS, err.Error())
			continue
		}
		if feed.Title == "" {
			log.Printf("Warning: feed from %s has no title", site.RSS)
			feed.Title = site.Name
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
		if len(items) > articlesPerSite {
			items = items[:articlesPerSite]
		}
		base, err := url.Parse(feed.UpdateURL)
		if err != nil {
			log.Fatal("failed parsing update URL of the feed")
		}
		feedLink, err := url.Parse(feed.Link)
		if err != nil {
			log.Fatal("failed parsing canonical feed URL of the feed")
		}
		for _, item := range items {
			rawSummary := item.Summary
			if len(rawSummary) == 0 {
				rawSummary = html.UnescapeString(item.Content)
			}
			summary := runewidth.Truncate(
				policy.Sanitize(rawSummary), summaryLength, "…")

			itemLink, err := url.Parse(item.Link)
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
	if len(articles) < articlesCount {
		articlesCount = len(articles)
	}
	articles = articles[:articlesCount]
	err = tmpl.Execute(os.Stdout, struct {
		Articles []*Article
	}{
		Articles: articles,
	})
	if err != nil {
		panic(err)
	}
}
