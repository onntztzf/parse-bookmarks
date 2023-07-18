package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// bookmark represents a bookmark entry with its title, URL, parent, and sub-bookmarks.
type Bookmark struct {
	Title     string     `json:"title"`
	URL       string     `json:"url,omitempty"`
	Parent    string     `json:"-"` // parent field is not included in JSON serialization.
	Bookmarks []Bookmark `json:"bookmarks,omitempty"`
	AddAt     *time.Time `json:"addAt,omitempty"`
	UpdateAt  *time.Time `json:"updateAt,omitempty"`
}

func main() {
	// read the HTML file containing the bookmarks data.
	htmlBytes, err := ioutil.ReadFile("bookmarks.html")
	if err != nil {
		fmt.Printf("error reading file: %s\n", err.Error())
		return
	}

	// parse the HTML using goquery library.
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(htmlBytes)))
	if err != nil {
		fmt.Printf("error parsing HTML: %s\n", err.Error())
		return
	}

	// extract bookmarks data from the HTML and create the bookmark tree.
	bookmarks := parseBookmarks(doc)
	tree := buildTree(bookmarks)

	// convert the bookmark tree to JSON and print the result.
	jsonData, err := json.Marshal(tree)
	if err != nil {
		fmt.Printf("error converting to JSON: %s\n", err.Error())
		return
	}
	fmt.Println(string(jsonData))
}

// parseBookmarks extracts bookmarks from the goquery document and returns a slice of bookmark entries.
func parseBookmarks(doc *goquery.Document) []Bookmark {
	// initialize a slice to store the extracted bookmarks.
	bookmarks := make([]Bookmark, 0, doc.Find("H3").Length())

	// iterate over each H3 element in the document representing bookmark titles.
	doc.Find("H3").Each(func(i int, header *goquery.Selection) {
		// create a bookmark entry for the current H3 element.
		bookmark := Bookmark{
			Title:    header.Text(),
			AddAt:    parseTime(header.AttrOr("add_date", "")),
			UpdateAt: parseTime(header.AttrOr("last_modified", "")),
		}

		// check if the header has a sibling DL element containing bookmarks.
		if dlNode := header.Next(); dlNode.Is("DL") {
			// initialize a slice to store the sub-bookmarks.
			bookmarks := make([]Bookmark, 0, dlNode.ChildrenFiltered("DT").Length())
			dlNode.ChildrenFiltered("DT").Each(func(j int, dtNode *goquery.Selection) {
				if aNode := dtNode.Children().First(); aNode.Is("A") {
					// create a bookmark entry for each bookmark within the DL element.
					bookmark := Bookmark{
						Title:    aNode.Text(),
						URL:      aNode.AttrOr("href", ""),
						AddAt:    parseTime(aNode.AttrOr("add_date", "")),
						UpdateAt: parseTime(aNode.AttrOr("last_modified", "")),
					}
					bookmarks = append(bookmarks, bookmark)
				}
			})
			// set the sub-bookmarks for the current bookmark.
			bookmark.Bookmarks = bookmarks
		}

		// check if the bookmark has a parent folder (H3 element).
		if parentDL := header.Parent().Parent(); parentDL.Is("DL") && parentDL.Prev().Is("H3") {
			// set the parent field for the current bookmark.
			bookmark.Parent = parentDL.Prev().Text()
		}

		// add the bookmark to the bookmarks slice.
		bookmarks = append(bookmarks, bookmark)
	})
	return bookmarks
}

// parseTime converts a timestamp string to a time.Time pointer.
func parseTime(timestamp string) *time.Time {
	if len(timestamp) == 0 {
		return nil
	}
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		fmt.Println("error parsing timestamp:", err.Error())
		return nil
	}
	t := time.Unix(ts, 0)
	return &t
}

// buildTree constructs the bookmark tree by finding the root folder and building the sub-trees.
func buildTree(bookmarks []Bookmark) Bookmark {
	// function to find the root folder by looking for a bookmark without a parent.
	findRootFolder := func(bookmarks []Bookmark) *Bookmark {
		for i := range bookmarks {
			if bookmarks[i].Parent == "" {
				return &bookmarks[i]
			}
		}
		return nil
	}

	// find the root folder and build the sub-trees recursively.
	root := findRootFolder(bookmarks)
	if root == nil {
		fmt.Println("root folder not found")
		return Bookmark{}
	}
	buildSubTree(root, bookmarks)

	return *root
}

// buildSubTree recursively builds the sub-tree under the parent bookmark.
func buildSubTree(parent *Bookmark, bookmarks []Bookmark) {
	for i := range bookmarks {
		if bookmarks[i].Parent == parent.Title {
			parent.Bookmarks = append(parent.Bookmarks, bookmarks[i])
			buildSubTree(&parent.Bookmarks[len(parent.Bookmarks)-1], bookmarks)
		}
	}
}
