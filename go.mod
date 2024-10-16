module github.com/kataras/iris/v12

go 1.23

retract [v12.0.0, v12.1.8] // Retract older versions as only latest is to be depended upon. Please update to @latest

require (
	github.com/BurntSushi/toml v1.4.0
	github.com/CloudyKit/jet/v6 v6.2.0
	github.com/Joker/jade v1.1.3
	github.com/Shopify/goreferrer v0.0.0-20240724165105-aceaa0259138
	github.com/andybalholm/brotli v1.1.1
	github.com/blang/semver/v4 v4.0.0
	github.com/dgraph-io/badger/v4 v4.3.1
	github.com/fatih/structs v1.1.0
	github.com/flosch/pongo2/v4 v4.0.2
	github.com/golang/snappy v0.0.4
	github.com/gomarkdown/markdown v0.0.0-20240930133441-72d49d9543d8
	github.com/google/uuid v1.6.0
	github.com/gorilla/securecookie v1.1.2
	github.com/iris-contrib/httpexpect/v2 v2.15.2
	github.com/iris-contrib/schema v0.0.6
	github.com/json-iterator/go v1.1.12
	github.com/kataras/blocks v0.0.8
	github.com/kataras/golog v0.1.12
	github.com/kataras/jwt v0.1.12
	github.com/kataras/neffos v0.0.24-0.20240927085411-23b9b4b5a867
	github.com/kataras/pio v0.0.14-0.20240707171706-2005199e2703
	github.com/kataras/sitemap v0.0.6
	github.com/kataras/tunnel v0.0.4
	github.com/klauspost/compress v1.17.11
	github.com/mailgun/raymond/v2 v2.0.48
	github.com/mailru/easyjson v0.7.7
	github.com/microcosm-cc/bluemonday v1.0.27
	github.com/redis/go-redis/v9 v9.6.2
	github.com/schollz/closestmatch v2.1.0+incompatible
	github.com/shirou/gopsutil/v3 v3.24.5
	github.com/tdewolff/minify/v2 v2.21.0
	github.com/vmihailenco/msgpack/v5 v5.4.1
	github.com/yosssi/ace v0.0.5
	go.etcd.io/bbolt v1.3.11
	golang.org/x/crypto v0.28.0
	golang.org/x/exp v0.0.0-20241009180824-f66d83c29e7c
	golang.org/x/net v0.30.0
	golang.org/x/sys v0.26.0
	golang.org/x/text v0.19.0
	golang.org/x/time v0.7.0
	google.golang.org/protobuf v1.35.1
	gopkg.in/ini.v1 v1.67.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/CloudyKit/fastprinter v0.0.0-20200109182630-33d98a066a53 // indirect
	github.com/ajg/form v1.5.1 // indirect
	github.com/aymerick/douceur v0.2.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgraph-io/ristretto v1.0.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/fatih/color v1.17.0 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/gobwas/httphead v0.1.0 // indirect
	github.com/gobwas/pool v0.2.1 // indirect
	github.com/gobwas/ws v1.4.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/flatbuffers v24.3.25+incompatible // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/gorilla/css v1.0.1 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/imkira/go-interpol v1.1.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/lufia/plan9stats v0.0.0-20240909124753-873cd0166683 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mediocregopher/radix/v3 v3.8.1 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/nats-io/nats.go v1.37.0 // indirect
	github.com/nats-io/nkeys v0.4.7 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/nxadm/tail v1.4.11 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/power-devops/perfstat v0.0.0-20240221224432-82ca36839d55 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/sanity-io/litter v1.5.5 // indirect
	github.com/sergi/go-diff v1.3.1 // indirect
	github.com/shoenig/go-m1cpu v0.1.6 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/stretchr/testify v1.9.0 // indirect
	github.com/tdewolff/parse/v2 v2.7.18 // indirect
	github.com/tklauser/go-sysconf v0.3.14 // indirect
	github.com/tklauser/numcpus v0.9.0 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/xeipuuv/gojsonschema v1.2.0 // indirect
	github.com/yalp/jsonpath v0.0.0-20180802001716-5cc68e5049a0 // indirect
	github.com/yudai/gojsondiff v1.0.0 // indirect
	github.com/yudai/golcs v0.0.0-20170316035057-ecda9a501e82 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	go.opencensus.io v0.24.0 // indirect
	golang.org/x/xerrors v0.0.0-20240903120638-7835f813f4da // indirect
	moul.io/http2curl/v2 v2.3.0 // indirect
)
