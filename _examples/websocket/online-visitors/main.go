package main

import (
	"fmt"
	"sync/atomic"

	"github.com/kataras/iris/v12"

	"github.com/kataras/iris/v12/websocket"
)

var events = websocket.Namespaces{
	"default": websocket.Events{
		websocket.OnRoomJoined: onRoomJoined,
		websocket.OnRoomLeft:   onRoomLeft,
	},
}

func main() {
	// init the web application instance
	// app := iris.New()
	app := iris.Default()

	// load templates
	app.RegisterView(iris.HTML("./templates", ".html").Reload(true))
	// setup the websocket server
	ws := websocket.New(websocket.DefaultGorillaUpgrader, events)

	app.Get("/my_endpoint", websocket.Handler(ws))

	// register static assets request path and system directory
	app.HandleDir("/js", iris.Dir("./static/assets/js"))

	h := func(ctx iris.Context) {
		ctx.ViewData("", page{PageID: "index page"})
		if err := ctx.View("index.html"); err != nil {
			ctx.HTML("<h3>%s</h3>", err.Error())
			return
		}
	}

	h2 := func(ctx iris.Context) {
		ctx.ViewData("", page{PageID: "other page"})
		if err := ctx.View("other.html"); err != nil {
			ctx.HTML("<h3>%s</h3>", err.Error())
			return
		}
	}

	// Open some browser tabs/or windows
	// and navigate to
	// http://localhost:8080/ and http://localhost:8080/other multiple times.
	// Each page has its own online-visitors counter.
	app.Get("/", h)
	app.Get("/other", h2)
	app.Listen(":8080")
}

type page struct {
	PageID string
}

type pageView struct {
	source string
	count  uint64
}

func (v *pageView) increment() {
	atomic.AddUint64(&v.count, 1)
}

func (v *pageView) decrement() {
	atomic.AddUint64(&v.count, ^uint64(0))
}

func (v *pageView) getCount() uint64 {
	return atomic.LoadUint64(&v.count)
}

type (
	pageViews []pageView
)

func (v *pageViews) Add(source string) {
	args := *v
	n := len(args)
	for i := 0; i < n; i++ {
		kv := &args[i]
		if kv.source == source {
			kv.increment()
			return
		}
	}

	c := cap(args)
	if c > n {
		args = args[:n+1]
		kv := &args[n]
		kv.source = source
		kv.count = 1
		*v = args
		return
	}

	kv := pageView{}
	kv.source = source
	kv.count = 1
	*v = append(args, kv)
}

func (v *pageViews) Get(source string) *pageView {
	args := *v
	n := len(args)
	for i := 0; i < n; i++ {
		kv := &args[i]
		if kv.source == source {
			return kv
		}
	}
	return nil
}

func (v *pageViews) Reset() {
	*v = (*v)[:0]
}

var v pageViews

func viewsCountBytes(viewsCount uint64) []byte {
	// * there are other methods to convert uint64 to []byte
	return []byte(fmt.Sprintf("%d", viewsCount))
}

func onRoomJoined(ns *websocket.NSConn, msg websocket.Message) error {
	// the roomName here is the source.
	pageSource := string(msg.Room)

	v.Add(pageSource)

	viewsCount := v.Get(pageSource).getCount()
	if viewsCount == 0 {
		viewsCount++ // count should be always > 0 here
	}

	// fire the "onNewVisit" client event
	// on each connection joined to this room (source page)
	// and notify of the new visit,
	// including this connection (see nil on first input arg).
	ns.Conn.Server().Broadcast(nil, websocket.Message{
		Namespace: msg.Namespace,
		Room:      pageSource,
		Event:     "onNewVisit", // fire the "onNewVisit" client event.
		Body:      viewsCountBytes(viewsCount),
	})

	return nil
}

func onRoomLeft(ns *websocket.NSConn, msg websocket.Message) error {
	// the roomName here is the source.
	pageV := v.Get(msg.Room)
	if pageV == nil {
		return nil // for any case that this room is not a pageView source
	}
	// decrement -1 the specific counter for this page source.
	pageV.decrement()

	// fire the "onNewVisit" client event
	// on each connection joined to this room (source page)
	// and notify of the new, decremented by one, visits count.
	ns.Conn.Server().Broadcast(nil, websocket.Message{
		Namespace: msg.Namespace,
		Room:      msg.Room,
		Event:     "onNewVisit",
		Body:      viewsCountBytes(pageV.getCount()),
	})

	return nil
}
