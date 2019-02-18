package main

import (
	"go.uber.org/atomic"
	"sync"
	"encoding/json"
	"strings"
)

type Profile struct {
	ConnectionId  string                   `json:"wid"`             //WsId - the temp user id
	UserLoginName string                   `json:"username"`        //user login name
	HashId        string                   `json:"h"`               //server key for this user to handling the hash
	PlayerId      int64                    `json:"i,omitempty"`     //sql user id
	TimeStamp     string                   `json:"t,omitempty"`     //the current time stamp
	SQLUid        int64                    `json:"uid,omitempty"`   //mysql user ID
	PhpToken      string                   `json:"token,omitempty"` //token to access php data static server
	walletx       []map[string]interface{} `json:"wallet"`          //a deeper level of the wallet
	inGameReg     atomic.Bool
	mSession      sync.RWMutex
	_mapWallet    *sync.Map
	// connectionLock sync.RWMutex
	sync.RWMutex
}


type Wallet struct {
	CoinId   int     `json:"i"`
	Currency string  `json:"c"`
	Balance  float64 `json:"b"`
	Locked   float64 `json:"l"`
}



func convert_list_profile(u *Profile) map[string]interface{} {
	return map[string]interface{}{
		"wid":      u.ConnectionId,
		"username": u.UserLoginName,
		"h":        u.HashId,
		"i":        u.PlayerId,
		"t":        u.TimeStamp,
		"uid":      u.SQLUid,
		"token":    u.PhpToken,
		"wallet":   fromSyncMapToArrayWallet(u._mapWallet),
	}
}


func convert_wallet(wallet *Wallet) map[string]interface{} {

	/*
		CoinId   int     `json:"i"`
		Currency string  `json:"c"`
		Balance  float64 `json:"b"`
		Locked   float64 `json:"l"`
	 */
	return map[string]interface{}{
		"i": wallet.CoinId,
		"c": strings.ToUpper(wallet.Currency),
		"b": wallet.Balance,
		"l": wallet.Locked,
	}
}

func fromSyncMapToArrayWallet(wallet_syncmap *sync.Map) []map[string]interface{} {
	var list_c []map[string]interface{}

	wallet_syncmap.Range(func(key, value interface{}) bool {
		list_c = append(list_c, convert_wallet(value.(*Wallet)))
		return true
	})

	return list_c
}

func report_login_result(user *Profile) []byte {
	out, er := json.Marshal(map[string]interface{}{
		"data":    convert_list_profile(user),
		"code":    1,
		"message": "success",
	})
	if er != nil {
		return []byte{}
	} else {
		return out
	}
}
