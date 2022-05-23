//go:build go1.18
// +build go1.18

package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kataras/iris/v12/auth"
)

type Provider struct {
	dataset []User

	invalidated    map[string]struct{} // key = token. Entry is blocked.
	invalidatedAll map[string]int64    // key = user id, value = timestamp. Issued before is consider invalid.
	mu             sync.RWMutex
}

func NewProvider() *Provider {
	return &Provider{
		dataset: []User{
			{
				ID:    "id-1",
				Email: "kataras2006@hotmail.com",
				Role:  Owner,
			},
			{
				ID:    "id-2",
				Email: "example@example.com",
				Role:  Member,
			},
		},
		invalidated:    make(map[string]struct{}),
		invalidatedAll: make(map[string]int64),
	}
}

func (p *Provider) Signin(ctx context.Context, username, password string) (User, error) { // fired on SigninHandler.
	// your database...
	for _, user := range p.dataset {
		if user.Email == username {
			return user, nil
		}
	}

	return User{}, fmt.Errorf("user not found")
}

func (p *Provider) ValidateToken(ctx context.Context, standardClaims auth.StandardClaims, u User) error { // fired on VerifyHandler.
	// your database and checks of blocked tokens...

	// check for specific token ids.
	p.mu.RLock()
	_, tokenBlocked := p.invalidated[standardClaims.ID]
	if !tokenBlocked {
		// this will disallow refresh tokens with origin jwt token id as the blocked access token as well.
		if standardClaims.OriginID != "" {
			_, tokenBlocked = p.invalidated[standardClaims.OriginID]
		}
	}
	p.mu.RUnlock()

	if tokenBlocked {
		return fmt.Errorf("token was invalidated")
	}
	//

	// check all tokens issuet before the "InvalidateToken" method was fired for this user.
	p.mu.RLock()
	ts, oldUserBlocked := p.invalidatedAll[u.ID]
	p.mu.RUnlock()

	if oldUserBlocked && standardClaims.IssuedAt <= ts {
		return fmt.Errorf("token was invalidated")
	}
	//

	return nil // else valid.
}

func (p *Provider) InvalidateToken(ctx context.Context, standardClaims auth.StandardClaims, u User) error { // fired on SignoutHandler.
	// invalidate this specific token.
	p.mu.Lock()
	p.invalidated[standardClaims.ID] = struct{}{}
	p.mu.Unlock()

	return nil
}

func (p *Provider) InvalidateTokens(ctx context.Context, u User) error { // fired on SignoutAllHandler.
	// invalidate all previous tokens came from "u".
	p.mu.Lock()
	p.invalidatedAll[u.ID] = time.Now().Unix()
	p.mu.Unlock()

	return nil
}
