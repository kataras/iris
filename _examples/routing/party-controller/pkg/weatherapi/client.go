package weatherapi

import (
	"context"
	"net/url"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/x/client"
)

// The BaseURL of our API client.
const BaseURL = "https://api.weatherapi.com/v1"

type (
	Options struct {
		APIKey string `json:"api_key" yaml:"APIKey" toml:"APIKey"`
	}

	Client struct {
		*client.Client
	}
)

func NewClient(opts Options) *Client {
	apiKeyParameterSetter := client.RequestParam("key", opts.APIKey)

	c := client.New(client.BaseURL(BaseURL),
		client.PersistentRequestOptions(apiKeyParameterSetter))

	return &Client{c}
}

func (c *Client) GetCurrentByCity(ctx context.Context, city string) (resp Response, err error) {
	urlpath := "/current.json"
	// ?q=Athens&aqi=no
	params := client.RequestQuery(url.Values{
		"q":   []string{city},
		"aqi": []string{"no"},
	})

	err = c.Client.ReadJSON(ctx, &resp, iris.MethodGet, urlpath, nil, params)
	return
}
