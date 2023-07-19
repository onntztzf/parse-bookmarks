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
	htmlBytes, err := ioutil.ReadFile("bookmarks_test1.html")
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
	// initialize a map to store bookmarks with their titles as keys.
	bookmarkMap := make(map[string]*Bookmark)

	// helper function to parse timestamp.
	parseTime := func(timestamp string) *time.Time {
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
			// iterate over each DT element representing sub-bookmark titles.
			dlNode.ChildrenFiltered("DT").Each(func(j int, dtNode *goquery.Selection) {
				if aNode := dtNode.Children().First(); aNode.Is("A") {
					// create a bookmark entry for each bookmark within the DL element.
					subBookmark := Bookmark{
						Title:    aNode.Text(),
						URL:      aNode.AttrOr("href", ""),
						AddAt:    parseTime(aNode.AttrOr("add_date", "")),
						UpdateAt: parseTime(aNode.AttrOr("last_modified", "")),
					}
					bookmark.Bookmarks = append(bookmark.Bookmarks, subBookmark)
				}
			})
		}

		// check if the bookmark has a parent folder (H3 element).
		if parentDL := header.Parent().Parent(); parentDL.Is("DL") && parentDL.Prev().Is("H3") {
			// set the parent field for the current bookmark.
			bookmark.Parent = parentDL.Prev().Text()
		}

		// add the bookmark to the map.
		bookmarkMap[bookmark.Title] = &bookmark
	})

	// convert the map values to a slice and return.
	bookmarks := make([]Bookmark, 0, len(bookmarkMap))
	for _, bookmark := range bookmarkMap {
		bookmarks = append(bookmarks, *bookmark)
	}
	return bookmarks
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

	root := findRootFolder(bookmarks)
	if root == nil {
		fmt.Println("root folder not found")
		return Bookmark{}
	}

	// function to build the sub-tree recursively.
	var buildSubTree func(parent *Bookmark)
	buildSubTree = func(parent *Bookmark) {
		for i := range bookmarks {
			if bookmarks[i].Parent == parent.Title {
				parent.Bookmarks = append(parent.Bookmarks, bookmarks[i])
				buildSubTree(&parent.Bookmarks[len(parent.Bookmarks)-1])
			}
		}
	}

	// build the sub-tree for the root folder.
	buildSubTree(root)
	return *root
}
