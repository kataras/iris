package geolocation

import (
	"context"
	"fmt"
	"net/http"

	"github.com/kataras/iris/v14/client"
)

// Info represents the geolocation information for an IP address.
type Info struct {
	Status      string  `json:"status"`
	Country     string  `json:"country"`
	CountryCode string  `json:"countryCode"`
	Region      string  `json:"region"`
	RegionName  string  `json:"regionName"`
	City        string  `json:"city"`
	Zip         string  `json:"zip"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	Timezone    string  `json:"timezone"`
	Isp         string  `json:"isp"`
	Org         string  `json:"org"`
	As          string  `json:"as"`
	Query       string  `json:"query"`
	Language    string  `json:"language"`
}

// fetchGeoLocationInfo fetches the geolocation information for the given IP address.
// It returns the geolocation information or an error if the request fails.
func fetchGeoLocationInfo(ctx context.Context, c *client.Client, ip string) (*Info, error) {
	url := "/" + ip

	geoLocation, err := c.BindJSON[Info](ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("error fetching data: %v", err)
	}

	if geoLocation.Status != "success" {
		return nil, fmt.Errorf("error: %s", geoLocation.Status)
	}

	return &geoLocation, nil
}
