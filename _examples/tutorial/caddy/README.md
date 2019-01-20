# Caddy loves Iris

The `Caddyfile` shows how you can use caddy to listen on ports 80 & 443 and sit in front of iris webserver(s) that serving on a different port (9091 and 9092 in this case; see Caddyfile).

## Running our two web servers

1. Go to `$GOPATH/src/github.com/kataras/iris/_examples/tutorial/caddy/server1`
2. Open a terminal window and execute `go run main.go`
3. Go to `$GOPATH/src/github.com/kataras/iris/_examples/tutorial/caddy/server2`
4. Open a new terminal window and execute `go run main.go`

## Caddy installation

1. Download caddy: https://caddyserver.com/download
2. Extract its contents where the `Caddyfile` is located, the `$GOPATH/src/github.com/kataras/iris/_examples/tutorial/caddy` in this case
3. Open, read and modify the `Caddyfile` to see by yourself how easy it is to configure the servers
4. Run `caddy` directly or open a terminal window and execute `caddy`
5. Go to `https://example.com` and `https://api.example.com/user/42`


## Notes

Iris has the `app.Run(iris.AutoTLS(":443", "example.com", "mail@example.com"))` which does
the exactly same thing but caddy is a great tool that helps you when you run multiple web servers from one host machine, i.e iris, apache, tomcat.