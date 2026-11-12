package oauth2

import (
	"fmt"
	"time"

	"github.com/kataras/iris/v14/rest/types"

	"golang.org/x/oauth2"
)

// DeviceCode holds information for the device code flow authorization.
type DeviceCode struct {
	// Code is the unique device code that the client will use to request a token.
	Code string `pg:"type=varchar(128),primary" json:"code"`
	// CreatedAt indicates when the code was generated.
	CreatedAt types.ISO8601 `pg:"type=timestamp,default=clock_timestamp()" json:"created_at,omitempty"`
	// UpdatedAt is the timestamp of last code update.
	UpdatedAt types.ISO8601 `pg:"type=timestamp,default=clock_timestamp()" json:"updated_at,omitempty"`
	// ExpiresAt is the time at which the device code will no longer be valid.
	ExpiresAt types.ISO8601 `pg:"type=timestamp" json:"expires_at"`
	// ClientID is the ID of the client requesting the device code.
	ClientID string `pg:"type=uuid,ref=oauth2_clients(id cascade),index" json:"client_id"`
	// UserData contains the authenticated user’s data.
	UserData AuthUserData `pg:"type=jsonb" json:"user_data"`
	// UserCode is the code displayed to the user, who must enter it on the verification URI.
	UserCode string `pg:"type=varchar(128)" json:"user_code"`
	// Scopes lists the permissions requested by the client.
	Scopes []string `pg:"type=varchar[]" json:"scopes"`
	// Interval is the polling interval in seconds for the client to check the device code.
	Interval int `pg:"type=int,default=0" json:"interval,omitzero"`
}

// NewDeviceCode creates and initializes a new DeviceCode with a unique device code
// and user code, to be used in the OAuth2 device flow.
func NewDeviceCode(clientID string, userData AuthUserData, interval int, scopes []string, maxAge int) (*DeviceCode, error) {
	// Generate device and user codes. The device code is machine-handled, so it
	// uses a high-entropy opaque token; the user code is typed by a human, so it
	// stays numeric but is widened from 3 to 8 digits (v14).
	code, err := generateToken(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate device code: %s: %w", clientID, err)
	}
	userCode, err := generateCode(8)
	if err != nil {
		return nil, fmt.Errorf("failed to generate user code: %s: %w", clientID, err)
	}

	now := Clock()

	deviceCode := &DeviceCode{
		Code:      code,
		CreatedAt: types.ISO8601(now),
		UpdatedAt: types.ISO8601(now),
		ExpiresAt: types.ISO8601(now.Add(time.Duration(maxAge) * time.Second)),
		ClientID:  clientID,
		UserData:  userData,
		UserCode:  userCode,
		Interval:  interval,
		Scopes:    scopes,
	}

	return deviceCode, nil
}

// HasExpired returns true if the current time is after the code’s ExpiresAt.
func (d *DeviceCode) HasExpired() bool {
	return Clock().After(d.ExpiresAt.ToTime())
}

// AuthResponse constructs and returns a device authorization response
// that includes the user code and verification URIs.
func (d *DeviceCode) AuthResponse(verificationURI string) oauth2.DeviceAuthResponse {
	var verificationURIComplete string
	if verificationURI != "" {
		verificationURIComplete = fmt.Sprintf("%s?user_code=%s", verificationURI, d.UserCode)
	}

	return oauth2.DeviceAuthResponse{
		DeviceCode:              d.Code,
		UserCode:                d.UserCode,
		VerificationURI:         verificationURI,
		VerificationURIComplete: verificationURIComplete,
		Expiry:                  d.ExpiresAt.ToTime(),
		Interval:                int64(d.Interval),
	}
}
