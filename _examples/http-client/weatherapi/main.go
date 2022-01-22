package main

import (
	"context"
	"fmt"

	"github.com/kataras/iris/v12/_examples/http-client/weatherapi/client"
)

func main() {
	c := client.NewClient(client.Options{
		APIKey: "{YOUR_API_KEY_HERE}",
	})

	resp, err := c.GetCurrentByCity(context.Background(), "Xanthi/GR")
	if err != nil {
		panic(err)
	}

	fmt.Printf("Temp: %.2f(C), %.2f(F)\n", resp.Current.TempC, resp.Current.TempF)
}
