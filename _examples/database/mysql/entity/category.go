package entity

import (
	"database/sql"
	"time"
)

// Category represents the categories entity.
// Each product belongs to a category, see `Product.CategoryID` field.
// It implements the `sql.Record` and `sql.Sorted` interfaces.
type Category struct {
	ID       int64  `db:"id" json:"id"`
	Title    string `db:"title" json:"title"`
	Position uint64 `db:"position" json:"position"`
	ImageURL string `db:"image_url" json:"image_url"`

	// We could use: sql.NullTime or unix time seconds (as int64),
	// note that the dsn parameter "parseTime=true" is required now in order to fill this field correctly.
	CreatedAt *time.Time `db:"created_at" json:"created_at"`
	UpdatedAt *time.Time `db:"updated_at" json:"updated_at"`
}

// TableName returns the database table name of a Category.
func (c *Category) TableName() string {
	return "categories"
}

// PrimaryKey returns the primary key of a Category.
func (c *Category) PrimaryKey() string {
	return "id"
}

// SortBy returns the column name that
// should be used as a fallback for sorting a set of Category.
func (c *Category) SortBy() string {
	return "position"
}

// Scan binds mysql rows to this Category.
func (c *Category) Scan(rows *sql.Rows) error {
	c.CreatedAt = new(time.Time)
	c.UpdatedAt = new(time.Time)
	return rows.Scan(&c.ID, &c.Title, &c.Position, &c.ImageURL, &c.CreatedAt, &c.UpdatedAt)
}

// Categories a list of categories. Implements the `Scannable` interface.
type Categories []*Category

// Scan binds mysql rows to this Categories.
func (cs *Categories) Scan(rows *sql.Rows) (err error) {
	cp := *cs
	for rows.Next() {
		c := new(Category)
		if err = c.Scan(rows); err != nil {
			return
		}
		cp = append(cp, c)
	}

	if len(cp) == 0 {
		return sql.ErrNoRows
	}

	*cs = cp

	return rows.Err()
}

/*
// The requests.
type (
	CreateCategoryRequest struct {
		Title    string `json:"title"` // all required.
		Position uint64 `json:"position"`
		ImageURL string `json:"imageURL"`
	}

	UpdateCategoryRequest CreateCategoryRequest // at least 1 required.

	GetCategoryRequest struct {
		ID int64 `json:"id"` // required.
	}

	DeleteCategoryRequest GetCategoryRequest

	GetCategoriesRequest struct {
		// [limit, offset...]
	}
)*/
