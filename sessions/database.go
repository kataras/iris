package sessions

import "time"

// Database is the interface which all session databases should implement
// By design it doesn't support any type of cookie session like other frameworks,
// I want to protect you, believe me, no context access (although we could)
// The scope of the database is to session somewhere the sessions in order to
//  keep them after restarting the server, nothing more.
// the values are sessions by the underline session, the check for new sessions, or
// 'this session value should added' are made automatically
// you are able just to set the values to your backend database with Load function.
// session database doesn't have any write or read access to the session, the loading of
// the initial data is done by the Load(string) (map[string]interfface{}, *time.Time) function
// synchronization are made automatically, you can register more than one session database
// but the first non-empty Load return data will be used as the session values.
// The Expire Date is given with data to save because the session entry must keep trace
// of the expire date in the case of the server is restarted. So the server will recover
// expiration state of session entry and it will track the expiration again.
// If expireDate is nil, that's means that there is no expire date.
type Database interface {
	Load(sid string) (datas map[string]interface{}, expireDate *time.Time)
	Update(sid string, datas map[string]interface{}, expireDate *time.Time)
}
