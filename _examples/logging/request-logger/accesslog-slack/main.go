package main

import (
	"os"
	"time"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/middleware/accesslog"
)

var (
	// https://api.slack.com/apps/your_app_id/oauth
	token = os.Getenv("SLACK_BOT_TOKEN")
	// on slack app: right click on the channel -> view channel details -> on bottom, copy the channel id.
	channelID = os.Getenv("SLACK_CHANNEL_ID")
)

// $ go run .
func main() {
	app := iris.New()

	ac := accesslog.New(os.Stdout) // or app.Logger().Printer
	ac.LatencyRound = time.Second
	ac.SetFormatter(&Slack{
		Token:         token,
		ChannelIDs:    []string{channelID},
		HandleMessage: true,
	})

	app.UseRouter(ac.Handler)
	app.Get("/", index)

	app.Listen(":8080")
}

func index(ctx iris.Context) {
	if sleepDur := ctx.URLParam("sleep"); sleepDur != "" {
		if d, err := time.ParseDuration(sleepDur); err == nil {
			time.Sleep(d)
		}
	}

	ctx.WriteString("Index")
}
