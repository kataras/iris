package oauth2

import (
	"html/template"

	"github.com/kataras/iris/v14/auth"
	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/context/encoding"

	"github.com/kataras/jwt"
)

// Clock returns the current server's time. By default it returns the time.Now().
// It can be overridden to return other time value.
//
// Usage: now := Clock()
var Clock = jwt.Clock

const (
	// DecisionApprove indicates the user approved the requested scopes.
	DecisionApprove = "approve"
	// DecisionDeny indicates the user denied the requested scopes.
	DecisionDeny = "deny"
)

// defaultConsentHandler returns a basic HTML form for the user to approve or deny consent.
// End-developers can replace it with a custom page if desired.
func defaultConsentHandler(errorHandler auth.ErrorHandler) context.Handler {
	var defaultConsentTemplate = template.Must(template.New("").Parse(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Confirm Access</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            background-color: #f4f4f4;
            margin: 0;
            padding: 0;
            display: flex;
            justify-content: center;
            align-items: center;
            height: 100vh;
        }
        .container {
            background-color: #fff;
            padding: 20px;
            border-radius: 8px;
            box-shadow: 0 0 10px rgba(0, 0, 0, 0.1);
            max-width: 500px;
            width: 100%;
        }
        h1 {
            font-size: 24px;
            margin-bottom: 20px;
        }
        p {
            font-size: 16px;
            margin-bottom: 20px;
        }
        ul {
            list-style-type: none;
            padding: 0;
            margin-bottom: 20px;
        }
        li {
            background-color: #f9f9f9;
            padding: 10px;
            border: 1px solid #ddd;
            border-radius: 4px;
            margin-bottom: 10px;
        }
        .client-info {
            display: flex;
            align-items: center;
            margin-bottom: 20px;
        }
        .client-info img {
            max-width: 50px;
            max-height: 50px;
            margin-right: 10px;
            border-radius: 4px;
        }
        .buttons {
            display: flex;
            justify-content: space-between;
        }
        .buttons button {
            padding: 10px 20px;
            border: none;
            border-radius: 4px;
            cursor: pointer;
            font-size: 16px;
        }
        .buttons button.approve {
            background-color: #4CAF50;
            color: white;
        }
        .buttons button.deny {
            background-color: #f44336;
            color: white;
        }
        .remember {
            display: flex;
            align-items: center;
            margin-bottom: 20px;
        }
        .remember input {
            margin-right: 10px;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>Confirm Access</h1>
        <form action="/oauth2/consent" method="post">
            <input type="hidden" name="client_id" value="{{.ClientID}}">
            <input type="hidden" name="state" value="{{.State}}">
            <input type="hidden" name="scope" value="{{.Scope}}">
            <div class="client-info">
                <img src="{{.ClientLogoURL}}" alt="Client Logo">
                <div>
                    <p>Client <strong>{{.ClientName}}</strong> (<a href="{{.ClientWebsite}}" target="_blank">{{.ClientWebsite}}</a>) is requesting access to the following scopes:</p>
                </div>
            </div>
            <ul>
                {{range .Scopes}}
                <li>{{.}}</li>
                {{end}}
            </ul>
            <div class="remember">
                <input type="checkbox" name="remember" value="1" checked>
                <label for="remember">Remember my decision for this client.</label>
            </div>
            <div class="buttons">
                <button type="submit" name="decision" value="allow" class="approve">Approve</button>
                <button type="submit" name="decision" value="deny" class="deny">Deny</button>
            </div>
        </form>
    </div>
</body>
</html>`))

	type Request struct { // coming from the /authorize (ask for consent page if ConsentPageURL is not empty).
		ClientID      string `url:"client_id" form:"client_id"`
		ClientName    string `url:"client_name" form:"client_name"`
		ClientWebsite string `url:"client_website" form:"client_website"`
		ClientLogoURL string `url:"client_logo_url" form:"client_logo_url"`
		State         string `url:"state" form:"state"`
		Scope         string `url:"scope" form:"scope"` // Scopes as single string.
	}

	decoder, err := encoding.NewQueryDecoder[Request](encoding.QueryDecoderOptions{})
	if err != nil {
		panic(err)
	}

	return func(ctx *context.Context) {
		var req Request
		if err := decoder.DecodeKeyValues(ctx.Request().URL.Query(), &req); err != nil {
			errorHandler.InvalidArgument(ctx, err)
			return
		}

		tmplData := struct {
			Request
			Scopes []string
		}{
			Request: req,
			Scopes:  SplitScopes(req.Scope),
		}

		ctx.SetContentType("text/html; charset=utf-8")
		if err := defaultConsentTemplate.Execute(ctx, tmplData); err != nil {
			ctx.StatusCode(500)
			ctx.WriteString(err.Error())
		}
	}
}
