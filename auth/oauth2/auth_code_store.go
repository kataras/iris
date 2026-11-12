package oauth2

import (
	"context"

	"github.com/kataras/iris/v14/core/pkg/cache"
)

// AuthCodeStore describes the actions needed for storing and retrieving OAuth2 authorization codes.
type AuthCodeStore interface {
	// InsertAuthCode saves a new authorization code into the store.
	InsertAuthCode(ctx context.Context, authCode *AuthCode) error
	// GetAndDeleteAuthCode retrieves and deletes the code from the store in one step.
	GetAndDeleteAuthCode(ctx context.Context, code string) (*AuthCode, bool, error)
}

// memAuthCodeStore provides an in-memory implementation of AuthCodeStore.
type memAuthCodeStore struct {
	auth cache.Cache[AuthCode]
}

// NewMemoryAuthCodeStore returns an AuthCodeStore backed by an in-memory cache.
func NewMemoryAuthCodeStore() AuthCodeStore {
	return &memAuthCodeStore{
		auth: cache.NewMemoryCache[AuthCode](0),
	}
}

// InsertAuthCode stores the authorization code in the in-memory cache.
func (s *memAuthCodeStore) InsertAuthCode(ctx context.Context, authCode *AuthCode) error {
	return s.auth.Set(ctx, authCode.Code, authCode)
}

// GetAndDeleteAuthCode retrieves the code and deletes it from the in-memory cache.
func (s *memAuthCodeStore) GetAndDeleteAuthCode(ctx context.Context, code string) (*AuthCode, bool, error) {
	authCode, ok, err := s.auth.Get(ctx, code)
	if ok {
		err = s.auth.Delete(ctx, code)
	}
	return authCode, ok, err
}

/* This will not work as client will not accept the server's Set-Cookie on redirects with < 300 status code.
// No need to have a single one for both codes, as the client either uses the auth code or the device code, not both.
// type codeStore struct{
// 	AuthCode *AuthCode[UserData]
// 	DeviceCode *DeviceCode
// }
// Other solution would be to have the Extra as a map["auth_code"](AuthCode) and ["device_code"]*DeviceCode,
// but again, the end-user only selects one way to authorize.

type cookieAuthCodeStore struct {
	sessiondb *sessions.SingleDatabase[UserData]
}

func NewAuthCodeCookieStore(sessiondb sessions.Database[UserData], cookieName string) AuthCodeStore[UserData] {
	gob.Register(&AuthCode[UserData]{})

	return &cookieAuthCodeStore[UserData]{
		sessiondb: sessions.Single(sessiondb, cookieName),
	}
}

func (s *cookieAuthCodeStore[UserData]) InsertAuthCode(ctx *context.Context, authCode *AuthCode[UserData]) error {
	session, err := s.sessiondb.Get(ctx)
	if err != nil {
		return err
	}

	session.SetExtra(authCode)
	return session.Commit(ctx)
}

func (s *cookieAuthCodeStore[UserData]) GetAndDeleteAuthCode(ctx *context.Context, code string) (*AuthCode[UserData], bool, error) {
	session, err := s.sessiondb.Get(ctx)
	if err != nil {
		return nil, false, err
	}

	if session.Extra != nil {
		if authCode, ok := session.Extra.(*AuthCode[UserData]); ok {
			if authCode.Code == code {
				session.Extra = nil
				session.Commit(ctx)
				return authCode, true, nil
			}
		}
	}

	return nil, false, nil
}
*/
