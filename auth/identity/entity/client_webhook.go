package entity

import "github.com/kataras/iris/v14/rest/types"

type WebhookActionType uint16

const (
	UserPasswordReset WebhookActionType = iota + 1
	UserDeleted                         // 2
)

// ClientWebhook represents a webhook for a client.
// The webhook is called when a specific action is triggered.
// Many webhooks can be registered for a client, even for the same actions.
type ClientWebhook struct {
	// ID is the primary key of the webhook.
	ID string `pg:"type=uuid,primary" json:"id,omitempty"`
	// CreatedAt is the timestamp when the webhook was created.
	CreatedAt types.ISO8601 `pg:"type=timestamp,default=clock_timestamp()" json:"created_at,omitempty"`
	// UpdatedAt is the timestamp when the webhook was last updated.
	// The updated_at is updated on each row update (see the trigger_set_timestamp sql function).
	UpdatedAt types.ISO8601 `pg:"type=timestamp,default=clock_timestamp()" json:"updated_at,omitempty"`
	// ClientID refers to the client that contains the webhook.
	ClientID string `pg:"type=uuid,ref=oauth2_clients(id cascade),index" json:"client_id"`
	// ActionType is the action type of webhook.
	ActionType WebhookActionType `pg:"type=smallint" json:"action_type"`
	// URL is the webhook URL. This URL is called (with any custom data depends to the ActionType) when the action is triggered.
	URL string `pg:"type=varchar(255)" json:"url"`
}
