package config

import (
	"github.com/imdario/mergo"
	"github.com/markbates/goth"
	"github.com/markbates/goth/providers/amazon"
	"github.com/markbates/goth/providers/bitbucket"
	"github.com/markbates/goth/providers/box"
	"github.com/markbates/goth/providers/digitalocean"
	"github.com/markbates/goth/providers/dropbox"
	"github.com/markbates/goth/providers/facebook"
	"github.com/markbates/goth/providers/github"
	"github.com/markbates/goth/providers/gitlab"
	"github.com/markbates/goth/providers/gplus"
	"github.com/markbates/goth/providers/heroku"
	"github.com/markbates/goth/providers/instagram"
	"github.com/markbates/goth/providers/lastfm"
	"github.com/markbates/goth/providers/linkedin"
	"github.com/markbates/goth/providers/onedrive"
	"github.com/markbates/goth/providers/paypal"
	"github.com/markbates/goth/providers/salesforce"
	"github.com/markbates/goth/providers/slack"
	"github.com/markbates/goth/providers/soundcloud"
	"github.com/markbates/goth/providers/spotify"
	"github.com/markbates/goth/providers/steam"
	"github.com/markbates/goth/providers/stripe"
	"github.com/markbates/goth/providers/twitch"
	"github.com/markbates/goth/providers/twitter"
	"github.com/markbates/goth/providers/uber"
	"github.com/markbates/goth/providers/wepay"
	"github.com/markbates/goth/providers/yahoo"
	"github.com/markbates/goth/providers/yammer"
)

const (
	// DefaultAuthPath /auth
	DefaultAuthPath = "/auth"
)

// OAuth the configs for the gothic oauth/oauth2 authentication for third-party websites
// All Key and Secret values are empty by default strings. Non-empty will be registered as Goth Provider automatically, by Iris
// the users can still register their own providers using goth.UseProviders
// contains the providers' keys  (& secrets) and the relative auth callback url path(ex: "/auth" will be registered as /auth/:provider/callback)
//
type OAuth struct {
	Path                                                  string
	TwitterKey, TwitterSecret, TwitterName                string
	FacebookKey, FacebookSecret, FacebookName             string
	GplusKey, GplusSecret, GplusName                      string
	GithubKey, GithubSecret, GithubName                   string
	SpotifyKey, SpotifySecret, SpotifyName                string
	LinkedinKey, LinkedinSecret, LinkedinName             string
	LastfmKey, LastfmSecret, LastfmName                   string
	TwitchKey, TwitchSecret, TwitchName                   string
	DropboxKey, DropboxSecret, DropboxName                string
	DigitaloceanKey, DigitaloceanSecret, DigitaloceanName string
	BitbucketKey, BitbucketSecret, BitbucketName          string
	InstagramKey, InstagramSecret, InstagramName          string
	BoxKey, BoxSecret, BoxName                            string
	SalesforceKey, SalesforceSecret, SalesforceName       string
	AmazonKey, AmazonSecret, AmazonName                   string
	YammerKey, YammerSecret, YammerName                   string
	OneDriveKey, OneDriveSecret, OneDriveName             string
	YahooKey, YahooSecret, YahooName                      string
	SlackKey, SlackSecret, SlackName                      string
	StripeKey, StripeSecret, StripeName                   string
	WepayKey, WepaySecret, WepayName                      string
	PaypalKey, PaypalSecret, PaypalName                   string
	SteamKey, SteamName                                   string
	HerokuKey, HerokuSecret, HerokuName                   string
	UberKey, UberSecret, UberName                         string
	SoundcloudKey, SoundcloudSecret, SoundcloudName       string
	GitlabKey, GitlabSecret, GitlabName                   string
}

// DefaultOAuth returns OAuth config, the fields of the iteral are zero-values ( empty strings)
func DefaultOAuth() OAuth {
	return OAuth{
		Path:             DefaultAuthPath,
		TwitterName:      "twitter",
		FacebookName:     "facebook",
		GplusName:        "gplus",
		GithubName:       "github",
		SpotifyName:      "spotify",
		LinkedinName:     "linkedin",
		LastfmName:       "lastfm",
		TwitchName:       "twitch",
		DropboxName:      "dropbox",
		DigitaloceanName: "digitalocean",
		BitbucketName:    "bitbucket",
		InstagramName:    "instagram",
		BoxName:          "box",
		SalesforceName:   "salesforce",
		AmazonName:       "amazon",
		YammerName:       "yammer",
		OneDriveName:     "onedrive",
		YahooName:        "yahoo",
		SlackName:        "slack",
		StripeName:       "stripe",
		WepayName:        "wepay",
		PaypalName:       "paypal",
		SteamName:        "steam",
		HerokuName:       "heroku",
		UberName:         "uber",
		SoundcloudName:   "soundcloud",
		GitlabName:       "gitlab",
	} // this will be registered as /auth/:provider in the mux
}

// MergeSingle merges the default with the given config and returns the result
func (c OAuth) MergeSingle(cfg OAuth) (config OAuth) {

	config = cfg
	mergo.Merge(&config, c)
	return
}

// GetAll returns the valid goth providers and the relative url paths (because the goth.Provider doesn't have a public method to get the Auth path...)
// we do the hard-core/hand checking here at the configs.
//
// receives one parameter which is the host from the server,ex: http://localhost:3000, will be used as prefix for the oauth callback
func (c OAuth) GetAll(vhost string) (providers []goth.Provider) {

	getCallbackURL := func(providerName string) string {
		return vhost + c.Path + "/" + providerName + "/callback"
	}

	//we could use a map but that's easier for the users because of code completion of their IDEs/editors
	if c.TwitterKey != "" && c.TwitterSecret != "" {
		println(getCallbackURL("twitter"))
		providers = append(providers, twitter.New(c.TwitterKey, c.TwitterSecret, getCallbackURL(c.TwitterName)))
	}
	if c.FacebookKey != "" && c.FacebookSecret != "" {
		providers = append(providers, facebook.New(c.FacebookKey, c.FacebookSecret, getCallbackURL(c.FacebookName)))
	}
	if c.GplusKey != "" && c.GplusSecret != "" {
		providers = append(providers, gplus.New(c.GplusKey, c.GplusSecret, getCallbackURL(c.GplusName)))
	}
	if c.GithubKey != "" && c.GithubSecret != "" {
		providers = append(providers, github.New(c.GithubKey, c.GithubSecret, getCallbackURL(c.GithubName)))
	}
	if c.SpotifyKey != "" && c.SpotifySecret != "" {
		providers = append(providers, spotify.New(c.SpotifyKey, c.SpotifySecret, getCallbackURL(c.SpotifyName)))
	}
	if c.LinkedinKey != "" && c.LinkedinSecret != "" {
		providers = append(providers, linkedin.New(c.LinkedinKey, c.LinkedinSecret, getCallbackURL(c.LinkedinName)))
	}
	if c.LastfmKey != "" && c.LastfmSecret != "" {
		providers = append(providers, lastfm.New(c.LastfmKey, c.LastfmSecret, getCallbackURL(c.LastfmName)))
	}
	if c.TwitchKey != "" && c.TwitchSecret != "" {
		providers = append(providers, twitch.New(c.TwitchKey, c.TwitchSecret, getCallbackURL(c.TwitchName)))
	}
	if c.DropboxKey != "" && c.DropboxSecret != "" {
		providers = append(providers, dropbox.New(c.DropboxKey, c.DropboxSecret, getCallbackURL(c.DropboxName)))
	}
	if c.DigitaloceanKey != "" && c.DigitaloceanSecret != "" {
		providers = append(providers, digitalocean.New(c.DigitaloceanKey, c.DigitaloceanSecret, getCallbackURL(c.DigitaloceanName)))
	}
	if c.BitbucketKey != "" && c.BitbucketSecret != "" {
		providers = append(providers, bitbucket.New(c.BitbucketKey, c.BitbucketSecret, getCallbackURL(c.BitbucketName)))
	}
	if c.InstagramKey != "" && c.InstagramSecret != "" {
		providers = append(providers, instagram.New(c.InstagramKey, c.InstagramSecret, getCallbackURL(c.InstagramName)))
	}
	if c.BoxKey != "" && c.BoxSecret != "" {
		providers = append(providers, box.New(c.BoxKey, c.BoxSecret, getCallbackURL(c.BoxName)))
	}
	if c.SalesforceKey != "" && c.SalesforceSecret != "" {
		providers = append(providers, salesforce.New(c.SalesforceKey, c.SalesforceSecret, getCallbackURL(c.SalesforceName)))
	}
	if c.AmazonKey != "" && c.AmazonSecret != "" {
		providers = append(providers, amazon.New(c.AmazonKey, c.AmazonSecret, getCallbackURL(c.AmazonName)))
	}
	if c.YammerKey != "" && c.YammerSecret != "" {
		providers = append(providers, yammer.New(c.YammerKey, c.YammerSecret, getCallbackURL(c.YammerName)))
	}
	if c.OneDriveKey != "" && c.OneDriveSecret != "" {
		providers = append(providers, onedrive.New(c.OneDriveKey, c.OneDriveSecret, getCallbackURL(c.OneDriveName)))
	}
	if c.YahooKey != "" && c.YahooSecret != "" {
		providers = append(providers, yahoo.New(c.YahooKey, c.YahooSecret, getCallbackURL(c.YahooName)))
	}
	if c.SlackKey != "" && c.SlackSecret != "" {
		providers = append(providers, slack.New(c.SlackKey, c.SlackSecret, getCallbackURL(c.SlackName)))
	}
	if c.StripeKey != "" && c.StripeSecret != "" {
		providers = append(providers, stripe.New(c.StripeKey, c.StripeSecret, getCallbackURL(c.StripeName)))
	}
	if c.WepayKey != "" && c.WepaySecret != "" {
		providers = append(providers, wepay.New(c.WepayKey, c.WepaySecret, getCallbackURL(c.WepayName)))
	}
	if c.PaypalKey != "" && c.PaypalSecret != "" {
		providers = append(providers, paypal.New(c.PaypalKey, c.PaypalSecret, getCallbackURL(c.PaypalName)))
	}
	if c.SteamKey != "" {
		providers = append(providers, steam.New(c.SteamKey, getCallbackURL(c.SteamName)))
	}
	if c.HerokuKey != "" && c.HerokuSecret != "" {
		providers = append(providers, heroku.New(c.HerokuKey, c.HerokuSecret, getCallbackURL(c.HerokuName)))
	}
	if c.UberKey != "" && c.UberSecret != "" {
		providers = append(providers, uber.New(c.UberKey, c.UberSecret, getCallbackURL(c.UberName)))
	}
	if c.SoundcloudKey != "" && c.SoundcloudSecret != "" {
		providers = append(providers, soundcloud.New(c.SoundcloudKey, c.SoundcloudSecret, getCallbackURL(c.SoundcloudName)))
	}
	if c.GitlabKey != "" && c.GitlabSecret != "" {
		providers = append(providers, gitlab.New(c.GitlabKey, c.GitlabSecret, getCallbackURL(c.GithubName)))
	}

	return
}
