package sdk

import (
	"context"
	"encoding/json"
	"net"
	"testing"

	"github.com/kataras/iris/v14/client"
	"github.com/kataras/iris/v14/rest/errors/validation"
	"github.com/kataras/iris/v14/rest/types"
)

func TestNew(t *testing.T) {
	// if testing.Short() {
	// 	t.Skip("skipping test in short mode")
	// }

	const baseURL = "http://localhost:8080"
	if _, err := net.Dial("tcp", baseURL); err != nil {
		t.Skipf("skipping test because the server is not running at %s", baseURL)
		return
	}

	type MyUser struct {
		ID        string           `json:"id,omitempty"`
		Username  string           `json:"username"`
		Firstname string           `json:"firstname"`
		LastName  string           `json:"lastname"`
		Email     string           `json:"email"`
		Birthdate types.SimpleDate `json:"birthdate"`
		Phone     string           `json:"phone"`
	}

	opts := Options{
		BaseURL:               baseURL,
		Token:                 "eyJraWQiOiJJUklTX0FVVEhfUkVGUkVTSCIsImFsZyI6IkVkRFNBIn0.Vv5tHtAp2uD9_I42wmLnqcQ4ckVHKMmi9PucK2R9eXbh8DoIxBZcGVvfCqjPbuPZYe2iZeYzqg5gsdkYvsPDCTogfPMGwveUOz-Y9J15N4kn0iJWAA1JPclcAf-zKWPKAPTGeUcnv3tYhx_SC4HQItvNUHwjROV87e76JWdrKPt4QtyOoxomZXgRhYnoqjRfHI0orx01z0NqASgsp5_iteAR2A6MdcEzQZQrOG8_2kD4y16qr4y8iOYC7PVk4l-8RHMOhteb7iCc9ojMyE1SSd6Gkilo5v446HM9yX0p0IJn3H3R02Wp3MIREEzf_rci7qCf10uUd-Ix_xumJ2bBhNcCt7kPoIQlut9m45_C8bVe_1Z89T4s51uPVuVJ1anfh15lU0NpbAD6Yk213ZldIYCGEL5o_961kow3V7byRihYYCdbimencO7Bbqz_K1umVTUSswG8TkY1AQaPn67uH5EM5s-EMFytrGyMvaeoWxf4ki2DdiHqmw_Fx-UyPosBfyFEIjZiMurAYcmkaV6RfGLujzWNYNj53yU0JoyQYh5k_WksgW622zINLhL0IaTwf_u1pF94nBD8hCmFg_hVD40uO_5SgdcZKf_sJTF9WxGUdufSNdMMTMrhYJiDxQwti8dT0M5rVsTXKYdIwkO9illsfCX7zZcslEp5csj4CxMQtlI.S6CJ8u7hIGH9IC0ptrKQA4OguW2Z-HDlBcBDTUWxhGtHioYwb7NnvyUGODtc8HMA8OZwobZ5jG3q3TgiRdJCCA",
		EncryptionKey:         "14be763a0cd7b7b98d48c3e808bfaf84",
		PasswordStrengthLevel: validation.PasswordMedium,
	}
	sdk, err := New[MyUser](opts)
	if err != nil {
		t.Fatal(err)
	}

	attrs, err := sdk.AdminGetUserSchema(context.Background())
	if err != nil {
		t.Fatalf("failed to get user schema: %v", err)
	}
	b, err := json.MarshalIndent(attrs, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal user schema: %v", err)
	}
	t.Logf("user schema: %s", string(b))

	// Sign up a new user.
	birthDate, _ := types.ParseSimpleDate("1993-01-01")
	_, err = sdk.AdminUserSignup(context.Background(), MyUser{
		Username:  "TeSt", // should be signed up as "test".
		Firstname: "Test",
		LastName:  "User",
		Email:     "makis@iris-go.com",
		Birthdate: birthDate,
		Phone:     "+306975807921",
	}, "Password")
	if err != nil {
		if client.GetErrorCode(err) != 409 { // not already exists.
			t.Fatalf("failed to signup user: %v", err)
		} else {
			t.Logf("user already exists...")
		}

		restored, err := sdk.AdminRestoreUser(context.Background(), AdminRestoreUserRequest{Identifier: "test"})
		if err != nil {
			t.Fatalf("failed to restore user: %v", err)
		}
		if !restored {
			t.Logf("user wasn't deleted, just already-existed")
		}

	} else {
		t.Logf("user signed-up")
	}

	ok, err := sdk.AdminResetUserPassword(context.Background(), AdminResetUserPasswordRequest{
		Identifier: "test",
		Password:   "NewPassword",
	})
	if err != nil {
		t.Fatalf("failed to reset user password: %v", err)
	}
	if !ok {
		t.Fatalf("request succeed but password didn't change at all")
	}

	tokenPair, err := sdk.UserSignin(context.Background(), "test", "NewPassword")
	if err != nil {
		t.Fatalf("failed to signin user: %v", err)
	}

	t.Logf("(signin) access token: %s", tokenPair.AccessToken)
	t.Logf("(signin) refresh token: %s", tokenPair.RefreshToken)

	tokenPair, err = sdk.AdminUserSignin(context.Background(), AdminUserSigninRequest{Identifier: "test"})
	if err != nil {
		t.Fatalf("failed to signin using admin (only username): %v", err)
	}

	t.Logf("(admin signin) access token: %s", tokenPair.AccessToken)
	t.Logf("(admin signin) refresh token: %s", tokenPair.RefreshToken)

	user, err := sdk.Introspect[MyUser](tokenPair.AccessToken)
	if err != nil {
		t.Fatalf("failed to introspect token locally: %v", err)
	}

	t.Logf("introspect token locally: %#+v", user)

	user2, err := sdk.TokenIntrospect[MyUser](context.Background(), tokenPair.AccessToken)
	if err != nil {
		t.Fatalf("failed to introspect token by API: %v", err)
	}

	t.Logf("introspect token from API: %#+v", user2)

	ok, err = sdk.AdminResetUserPassword(context.Background(), AdminResetUserPasswordRequest{
		Identifier: "test",
		Password:   "Password", // reset back to the original password.
	})
	if err != nil {
		t.Fatalf("failed to reset user password: %v", err)
	}
	if !ok {
		t.Fatalf("request succeed but password didn't change at all")
	}

	deleted, err := sdk.AdminDeleteUser(context.Background(), AdminDeleteUserRequest{Identifier: user.ID, Soft: true})
	if err != nil {
		t.Fatalf("failed to delete user: %v", err)
	}

	if !deleted {
		t.Fatalf("failed to delete user: %#+v", user)
	}
}
