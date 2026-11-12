package entity

import "github.com/kataras/iris/v14/rest/types"

type UserGroup struct {
	// ID is the primary key of the user group .
	ID string `pg:"type=uuid,primary" json:"id,omitempty"`
	// CreatedAt is the timestamp when the user group was created.
	CreatedAt types.ISO8601 `pg:"type=timestamp,default=clock_timestamp()" json:"created_at,omitempty"`
	// UpdatedAt is the timestamp when the user group was last updated.
	// The updated_at is updated on each row update (see the trigger_set_timestamp sql function).
	UpdatedAt types.ISO8601 `pg:"type=timestamp,default=clock_timestamp()" json:"updated_at,omitempty"`
	// Name is the name of the user group.
	Name string `pg:"type=varchar(255),unique" json:"name,omitempty"`
	// Filter is a JSONB field that contains filter options for the user group.
	Filter UserFilterOptions `pg:"type=jsonb,default='{}'" json:"filter"`
}
