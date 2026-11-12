package geolocation

import (
	"context"
	"fmt"
	"net/http"

	"github.com/kataras/iris/v14/client"
)

type countryInfo struct {
	Languages map[string]string `json:"languages"`
}

// fetchLanguage fetches the language for the given country code.
// It returns the language or an error if the request fails.
func fetchLanguage(ctx context.Context, c *client.Client, countryCode string) (string, error) {
	languageURL := "/" + countryCode

	countryInfo, err := c.BindJSON[[]countryInfo](ctx, http.MethodGet, languageURL, nil)
	if err != nil {
		return "", fmt.Errorf("error fetching language data: %v", err)
	}

	if len(countryInfo) > 0 && len(countryInfo[0].Languages) > 0 {
		for _, language := range countryInfo[0].Languages {
			return language, nil
		}
	}

	return "", nil
}
