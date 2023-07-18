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

// bookmark represents a bookmark entry in the tree.
type Bookmark struct {
	Title     string     `json:"title"`
	URL       string     `json:"url,omitempty"`
	Parent    string     `json:"-"`
	Bookmarks []Bookmark `json:"bookmarks,omitempty"`
	AddAt     *time.Time `json:"addAt,omitempty"`
	UpdateAt  *time.Time `json:"updateAt,omitempty"`
}

func main() {
	// read the HTML file containing bookmarks data
	htmlBytes, err := ioutil.ReadFile("bookmarks.html")
	if err != nil {
		fmt.Printf("Error reading file: %s\n", err.Error())
		return
	}

	// parse the HTML content using goquery
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(htmlBytes)))
	if err != nil {
		fmt.Printf("Error parsing HTML: %s\n", err.Error())
		return
	}

	// parse the bookmarks data from the HTML
	bookmarks := parseBookmarks(doc)

	// build the tree structure from the parsed bookmarks
	tree := buildTree(bookmarks)

	// convert the tree to JSON format
	jsonData, err := json.Marshal(tree)
	if err != nil {
		fmt.Printf("Error converting to JSON: %s\n", err.Error())
		return
	}

	// print the JSON data
	fmt.Println(string(jsonData))
}

// parseBookmarks extracts bookmarks from the goquery document and returns them as a slice of Bookmark objects.
func parseBookmarks(doc *goquery.Document) []Bookmark {
	bookmarks := make([]Bookmark, 0, doc.Find("H3").Length())
	doc.Find("H3").Each(func(i int, header *goquery.Selection) {
		bookmark := Bookmark{
			Title:    header.Text(),
			AddAt:    parseTime(header.AttrOr("add_date", "")),
			UpdateAt: parseTime(header.AttrOr("last_modified", "")),
		}

		// Find the DL element containing bookmark entries
		if dlNode := header.Next(); dlNode.Is("DL") {
			bookmarks := make([]Bookmark, 0, dlNode.ChildrenFiltered("DT").Length())

			dlNode.ChildrenFiltered("DT").Each(func(j int, dtNode *goquery.Selection) {
				if aNode := dtNode.Children().First(); aNode.Is("A") {
					bookmark := Bookmark{
						Title:    aNode.Text(),
						URL:      aNode.AttrOr("href", ""),
						AddAt:    parseTime(aNode.AttrOr("add_date", "")),
						UpdateAt: parseTime(aNode.AttrOr("last_modified", "")),
					}
					bookmarks = append(bookmarks, bookmark)
				}
			})
			bookmark.Bookmarks = bookmarks
		}

		// Find the parent folder title if it exists
		if parentDL := header.Parent().Parent(); parentDL.Is("DL") && parentDL.Prev().Is("H3") {
			bookmark.Parent = parentDL.Prev().Text()
		}
		bookmarks = append(bookmarks, bookmark)
	})
	return bookmarks
}

// parseTime converts ADD_DATE attribute value to time.Time.
func parseTime(timestamp string) *time.Time {
	if len(timestamp) == 0 {
		return nil
	}
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		fmt.Println("Error parsing timestamp:", err.Error())
		return nil
	}
	t := time.Unix(ts, 0)
	return &t
}

// buildTree constructs the tree of bookmarks from the flat bookmarks slice.
func buildTree(bookmarks []Bookmark) Bookmark {
	root := findRootFolder(bookmarks)
	if root == nil {
		fmt.Println("Root folder not found")
		return Bookmark{}
	}
	buildSubTree(root, bookmarks)

	return *root
}

// findRootFolder finds and returns the root folder in the bookmarks slice.
func findRootFolder(bookmarks []Bookmark) *Bookmark {
	for i := range bookmarks {
		if bookmarks[i].Parent == "" {
			return &bookmarks[i]
		}
	}
	return nil
}

// buildSubTree recursively builds the subtree of bookmarks starting from the given parent folder.
func buildSubTree(parent *Bookmark, bookmarks []Bookmark) {
	for i := range bookmarks {
		if bookmarks[i].Parent == parent.Title {
			parent.Bookmarks = append(parent.Bookmarks, bookmarks[i])
			buildSubTree(&parent.Bookmarks[len(parent.Bookmarks)-1], bookmarks)
		}
	}
}
