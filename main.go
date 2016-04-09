package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/net/html"
)

const baseURL = "https://digital.darkhorse.com/browse/all/"
const maxRequests = 10

var errNoPages = errors.New("Could not find any page in the comic book catalog")
var errInvalidNumberOfPages = errors.New("Invalid number of pages in the comic book catalog")

func parseHTML(url string) (*html.Node, error) {
	response, err := http.Get(url)

	if err != nil {
		return nil, err
	}

	defer response.Body.Close()

	return html.Parse(response.Body)
}

func findPageSelectElement(n *html.Node) *html.Node {
	if n.Type == html.ElementNode && n.Data == "select" {
		for _, attr := range n.Attr {
			if attr.Key == "id" && attr.Val == "page-select" {
				return n
			}
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if pageSelect := findPageSelectElement(c); pageSelect != nil {
			return pageSelect
		}
	}

	return nil
}

func getNumberOfPagesFromElement(n *html.Node) (int, error) {
	var c *html.Node

	for c = n.LastChild; c != nil; c = c.PrevSibling {
		if c.Type == html.ElementNode && c.Data == "option" {
			break
		}
	}

	if c == nil {
		return 0, errInvalidNumberOfPages
	}

	lastPage := c.FirstChild

	if lastPage == nil || lastPage.Type != html.TextNode {
		return 0, errInvalidNumberOfPages
	}

	count, err := strconv.Atoi(lastPage.Data)

	if err != nil || count <= 0 {
		return 0, errInvalidNumberOfPages
	}

	return count, nil
}

func getNumberOfPages(url string) (int, error) {
	doc, err := parseHTML(url)

	if err != nil {
		return 0, err
	}

	pageSelectElement := findPageSelectElement(doc)

	if pageSelectElement == nil {
		return 0, errNoPages
	}

	return getNumberOfPagesFromElement(pageSelectElement)
}

func getAttributes(n *html.Node) map[string]string {
	attributes := make(map[string]string)

	for _, attr := range n.Attr {
		attributes[attr.Key] = attr.Val
	}

	return attributes
}

func getNames(url string) []string {
	var names []string

	if doc, err := parseHTML(url); err == nil {
		var f func(n *html.Node)

		f = func(n *html.Node) {
			if n.Type == html.ElementNode && n.Data == "a" {
				attr := getAttributes(n)
				class, title := attr["class"], attr["title"]

				if strings.HasPrefix(class, "cover ") && title != "" {
					names = append(names, title)
					return
				}
			}

			for c := n.FirstChild; c != nil; c = c.NextSibling {
				f(c)
			}
		}

		f(doc)
	}

	return names
}

func main() {
	count, err := getNumberOfPages(baseURL)

	if err != nil {
		log.Fatal(err)
	}

	var wg sync.WaitGroup
	wg.Add(maxRequests)

	urls := make(chan string)
	names := make(chan string)
	result := make(chan []string)

	for i := 0; i < maxRequests; i++ {
		go func() {
			defer wg.Done()

			for url := range urls {
				for _, name := range getNames(url) {
					names <- name
				}
			}
		}()
	}

	go func() {
		var sorted []string

		for name := range names {
			sorted = append(sorted, name)
		}

		sort.Strings(sorted)
		result <- sorted
	}()

	for i := 0; i < count; i++ {
		urls <- fmt.Sprintf("%s?page=%d", baseURL, i+1)
	}

	close(urls)
	wg.Wait()
	close(names)

	for _, name := range <-result {
		fmt.Println(name)
	}
}
