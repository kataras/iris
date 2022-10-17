package main

import (
	"io"

	"github.com/kataras/iris/v12/middleware/accesslog"

	"github.com/slack-go/slack"
)

type Slack struct {
	// Client is the underline slack slack.
	// This or Token are required.
	Client *slack.Client
	// Token is the oauth slack client token.
	// Read more at: https://api.slack.com/web#authentication.
	//
	// If Client is not null then this is used to initialize the Slack.Client field.
	Token string

	// ChannelIDs specifies one or more channel the request logs
	// will be printed to.
	ChannelIDs []string

	// HandleMessage set to true whether the slack formatter
	// should just send the log to the slack channel(s) and
	// stop printing the log to the accesslog's io.Writer output.
	HandleMessage bool

	// Template is the underline text template format of the logs.
	// Set to a custom one if you want to customize the template (how the text is rended).
	Template *accesslog.Template
}

func (f *Slack) SetOutput(dest io.Writer) {
	if f.Client == nil && f.Token == "" {
		panic("client or token fields must be provided")
	}

	if len(f.ChannelIDs) == 0 {
		panic("channel ids field is required")
	}

	if f.Token != "" {
		c := slack.New(f.Token)
		f.Client = c
	}

	if f.Template == nil {
		f.Template = &accesslog.Template{}
	}

	f.Template.SetOutput(dest)
}

func (f *Slack) Format(log *accesslog.Log) (bool, error) {
	text, err := f.Template.LogText(log)
	if err != nil {
		return false, err
	}

	for _, channelID := range f.ChannelIDs {
		_, _, err := f.Client.PostMessage(
			channelID,
			slack.MsgOptionText(text, false),
			slack.MsgOptionAsUser(true),
		)

		if err != nil {
			return false, err
		}
	}

	return f.HandleMessage, nil
}
