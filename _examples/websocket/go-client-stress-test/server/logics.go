package main

import (
	// _ "github.com/kataras/iris/websocket2"
	"../../../../ws1m"
	"time"
	"sync"
	"github.com/Pallinder/sillyname-go"
	"github.com/shomali11/util/xhashes"
)

func badlogictesting(ws *websocket.Server) {
	for {
		time.Sleep(500 * time.Millisecond)
		sendall(ws)
	}
}
func simulate_ping(ws *websocket.Server) {
	for {
		time.Sleep(35500 * time.Millisecond)
		sendallp(ws)
	}
}

func sendallp(ws *websocket.Server) {

	for _, conn := range ws.GetConnections() {
		conn.Emit("test-ping-pong", string("some json in here"))
	}
}

func sendall(ws *websocket.Server) {
	for _, conn := range ws.GetConnections() {
		conn.Emit("test-c", string("some json in here"))
	}
}

func generateUser(c websocket.Connection) {

	user := &Profile{}

	time.Sleep(500 * time.Millisecond)
	// long operations

	Nick_name := sillyname.GenerateStupidName()
	Hash_ID := xhashes.SHA512(Nick_name)
	//u1uuid := uuid.Must(uuid.NewV4())
	wallets := CreateWalletWithCoin(0.00)
	//	uuid := nuPerson(db, Nick_name, Hash_ID, temp_user)
	//temp user id
	user.ConnectionId = c.ID()
	//the display player name
	user.UserLoginName = Nick_name
	//the hash ID
	user.HashId = Hash_ID
	//the wallet
	user._mapWallet = wallets

	c.Emit("start", string(report_login_result(user)))
}



func CreateWalletWithCoin(k float64) *sync.Map {
	return &sync.Map{}
}

