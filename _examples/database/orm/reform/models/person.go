//go:generate reform
package models

import "time"

//reform:people
type Person struct {
	ID        int32      `reform:"id,pk" json:"id"`
	Name      string     `reform:"name" json:"name"`
	Email     *string    `reform:"email" json:"email"`
	CreatedAt time.Time  `reform:"created_at" json:"created_at"`
	UpdatedAt *time.Time `reform:"updated_at" json:"updated_at"`
}
