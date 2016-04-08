package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"golang.org/x/net/html"
)

const url = "https://digital.darkhorse.com/browse/all/"
const maxRequests = 10

var errNoPages = errors.New("Could not find any page in the comic book catalog")
var errInvalidNumberOfPages = errors.New("Invalid number of pages in the comic book catalog")

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
	response, err := http.Get(url)

	if err != nil {
		return 0, err
	}

	defer response.Body.Close()

	doc, err := html.Parse(response.Body)

	if err != nil {
		return 0, err
	}

	pageSelectElement := findPageSelectElement(doc)

	if pageSelectElement == nil {
		return 0, errNoPages
	}

	return getNumberOfPagesFromElement(pageSelectElement)
}

func main() {
	count, err := getNumberOfPages(url)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(count)
}
