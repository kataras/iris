package examples

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/PuerkitoBio/goquery"
	"github.com/kataras/iris/core/errors"
	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday"
)

// we could directly query and parse the github page for _examples and take
// the examples from its folders, without even the need of a readme to be exist. But I will not do that
// because github may change its structure to these folders, so its safer to just:
// convert the raw readme.md to the html
// query the new html and parse its ul and li tags,
// markdown syntax for these things will (never) change, so I assume it will work for a lot of years.
const (
	branch = "master"
	// rootURL = "https://github.com/kataras/iris/tree/" + branch + "/_examples"
	// rawRootURL = "https://raw.githubusercontent.com/kataras/iris/"+branch"/_examples/"
	contentsURL      = "https://raw.githubusercontent.com/kataras/iris/" + branch + "/_examples/README.md"
	tableOfContents  = "Table of contents"
	sanitizeMarkdown = true
)

// WriteExamplesTo will write all examples to the "w"
func WriteExamplesTo(w io.Writer) (categories []Category, err error) {
	// if len(categoryName) == 0 {
	// 	return nil, errors.New("category is empty")
	// }
	// categoryName = strings.ToTitle(categoryName) // i.e Category Name

	// category := Category{
	// 	Name: categoryName,
	// }

	// get the raw markdown
	res, err := http.Get(contentsURL)
	if err != nil {
		return nil, err
	}
	markdownContents, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	// convert it to html
	htmlContents := &bytes.Buffer{}
	htmlContentsFromMarkdown := blackfriday.MarkdownCommon(markdownContents)

	if len(htmlContentsFromMarkdown) == 0 {
		return nil, errors.New("empty html")
	}

	if sanitizeMarkdown {
		markdownContents = bluemonday.UGCPolicy().SanitizeBytes(markdownContents)
	}

	htmlContents.Write(htmlContentsFromMarkdown)
	// println("html contents: " + htmlContents.String())
	// get the document from the html
	readme, err := goquery.NewDocumentFromReader(htmlContents)
	if err != nil {
		return nil, err
	}

	// or with just one line (but may break if we add another h2, so I will do it with the hard and un-readable way for now)
	// readme.Find("h2").First().NextAllFiltered("ul").Children().Text()

	// find the header of Table Of Contents, we will need it to take its
	// next ul, which should be the examples list.
	var tableOfContentsHeader *goquery.Selection
	readme.Find("h2").EachWithBreak(func(_ int, n *goquery.Selection) bool {
		if nodeContents := n.Text(); nodeContents == tableOfContents {
			tableOfContentsHeader = n
			return false // break
		}
		return true
	})

	// println(tableOfContentsHeader.Text())

	if tableOfContentsHeader == nil {
		return nil, errors.New("table of contents not found using: " + tableOfContents)
	}

	// get the list of the examples
	tableOfContentsUL := tableOfContentsHeader.NextAllFiltered("ul")
	if tableOfContentsUL == nil {
		return nil, errors.New("table of contents list not found")
	}

	// iterate over categories example's <a href ...>...</a>
	tableOfContentsUL.Children().EachWithBreak(func(_ int, li *goquery.Selection) bool {
		exampleHrefLink := li.Children().First()
		if exampleHrefLink == nil {
			err = errors.New("example link href is nil, source: " + li.Text())
			return false // break on first failure
		}

		categoryName := exampleHrefLink.Text()

		println(categoryName)

		category := Category{
			Name: categoryName,
		}

		_ = category

		li.Find("ul").Children().Each(func(_ int, liExample *goquery.Selection) {
			println(liExample.Text())
		})

		return true
	})

	return nil, err
}
