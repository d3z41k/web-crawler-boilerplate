package main

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"net/http"
	"os"
	"strings"
)

const MaxDepth = 2

type Link struct {
	url   string
	text  string
	depth int
}

type HttpError struct {
	original string
}

func LinkReader(resp *http.Response, depth int) []Link {
	page := html.NewTokenizer(resp.Body)

	var links []Link
	var start *html.Token
	var text string

	for {
		_ = page.Next()
		token := page.Token()
		if token.Type == html.ErrorToken {
			break
		}

		if start != nil && token.Type == html.TextToken {
			text = fmt.Sprintf("%s%s", text, token.Data)
		}

		if token.DataAtom == atom.A {
			switch token.Type {
			case html.StartTagToken:
				if len(token.Attr) > 0 {
					start = &token
				}
			case html.EndTagToken:
				if start == nil {
					log.Warnf("Link End found without start: %s", text)
					continue
				}
				link := NewLink(*start, text, depth)
				if link.Valid() {
					links = append(links, link)
					log.Debugf("Link found %s", link)
				}

				start = nil
				text = ""

			}
		}
	}

	log.Debug(links)

	return links
}

func NewLink(tag html.Token, text string, depth int) Link {
	link := Link{text: strings.TrimSpace(text), depth: depth}

	for i := range tag.Attr {
		if tag.Attr[i].Key == "href" {
			link.url = strings.TrimSpace(tag.Attr[i].Val)
		}
	}
	return link
}

func (l *Link) String() string {
	spacer := strings.Repeat("\t", l.depth)

	return fmt.Sprintf("%s%s (%d) - %s", spacer, l.text, l.depth, l.url)
}

func (l *Link) Valid() bool {
	if l.depth >= MaxDepth {
		return false
	}

	if len(l.text) == 0 {
		return false
	}

	if len(l.url) == 0 || strings.Contains(strings.ToLower(l.url), "javascript") {
		return false
	}

	return true
}

func (l HttpError) Error() string {
	return l.original
}

func recurDownloader(url string, depth int) {
	page, err := downloader(url)

	if err != nil {
		log.Error(err)
		return
	}

	links := LinkReader(page, depth)

	for _, link := range links {
		fmt.Println(link)
		if depth+1 < MaxDepth {
			recurDownloader(link.url, depth+1)
		}
	}
}

func downloader(url string) (resp *http.Response, err error) {
	log.Debugf("Downloading %s", url)
	resp, err = http.Get(url)

	if err != nil {
		log.Debugf("Error: %s", err)
		return
	}

	if resp.StatusCode > 299 {
		err = HttpError{fmt.Sprintf("Error (%d): %s", resp.StatusCode, url)}
		log.Debug(err)
		return
	}
	return
}

func main() {
	log.SetLevel(log.InfoLevel)

	log.Debug(os.Args)

	if len(os.Args) < 2 {
		log.Fatalln("Missing Url argument")
	}

	recurDownloader(os.Args[1], 0)
}
