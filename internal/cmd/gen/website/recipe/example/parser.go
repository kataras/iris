package example

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/kataras/iris/core/errors"
	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday"
)

// Parse will try to parse and return all examples.
// The input parameter "branch" is used to build
// the raw..iris-contrib/examples/$branch/
// but it should be the same with
// the kataras/iris/$branch/ for consistency.
func Parse(branch string) (examples []Example, err error) {
	var (
		contentsURL      = "https://raw.githubusercontent.com/iris-contrib/examples/" + branch
		tableOfContents  = "Table of contents"
		sanitizeMarkdown = true
	)

	// get the raw markdown
	readmeURL := contentsURL + "/README.md"
	res, err := http.Get(readmeURL)
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

	// or with just one line (but may break if we add another h2,
	// so I will do it with the hard and un-readable way for now)
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

	if tableOfContentsHeader == nil {
		return nil, errors.New("table of contents not found using: " + tableOfContents)
	}

	// get the list of the examples
	tableOfContentsUL := tableOfContentsHeader.NextFiltered("ul")
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

		name := exampleHrefLink.Text()

		sourcelink, _ := li.Find("a").First().Attr("href")
		hasChildren := !strings.HasSuffix(sourcelink, ".go")

		example := Example{
			Name:           name,
			DataSource:     contentsURL + "/" + sourcelink,
			HasChildren:    hasChildren,
			HasNotChildren: !hasChildren,
		}

		// search for sub examples
		if hasChildren {
			li.Find("ul").Children().EachWithBreak(func(_ int, liExample *goquery.Selection) bool {
				name := liExample.Text()
				liHref := liExample.Find("a").First()
				sourcelink, ok := liHref.Attr("href")
				if !ok {
					err = errors.New(name + "'s source couldn't be found")
					return false
				}

				subExample := Example{
					Name:       name,
					DataSource: contentsURL + "/" + sourcelink,
				}

				example.Children = append(example.Children, subExample)
				return true
			})

		}

		examples = append(examples, example)
		return true
	})
	return examples, err
}
