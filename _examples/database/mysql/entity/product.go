package entity

import (
	"database/sql"
	"time"
)

// Product represents the products entity.
// It implements the `sql.Record` and `sql.Sorted` interfaces.
type Product struct {
	ID          int64      `db:"id" json:"id"`
	CategoryID  int64      `db:"category_id" json:"category_id"`
	Title       string     `db:"title" json:"title"`
	ImageURL    string     `db:"image_url" json:"image_url"`
	Price       float32    `db:"price" json:"price"`
	Description string     `db:"description" json:"description"`
	CreatedAt   *time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   *time.Time `db:"updated_at" json:"updated_at"`
}

// TableName returns the database table name of a Product.
func (p Product) TableName() string {
	return "products"
}

// PrimaryKey returns the primary key of a Product.
func (p *Product) PrimaryKey() string {
	return "id"
}

// SortBy returns the column name that
// should be used as a fallback for sorting a set of Product.
func (p *Product) SortBy() string {
	return "updated_at"
}

// ValidateInsert simple check for empty fields that should be required.
func (p *Product) ValidateInsert() bool {
	return p.CategoryID > 0 && p.Title != "" && p.ImageURL != "" && p.Price > 0 /* decimal* */ && p.Description != ""
}

// Scan binds mysql rows to this Product.
func (p *Product) Scan(rows *sql.Rows) error {
	p.CreatedAt = new(time.Time)
	p.UpdatedAt = new(time.Time)
	return rows.Scan(&p.ID, &p.CategoryID, &p.Title, &p.ImageURL, &p.Price, &p.Description, &p.CreatedAt, &p.UpdatedAt)
}

// Products is a list of products. Implements the `Scannable` interface.
type Products []*Product

// Scan binds mysql rows to this Categories.
func (ps *Products) Scan(rows *sql.Rows) (err error) {
	cp := *ps
	for rows.Next() {
		p := new(Product)
		if err = p.Scan(rows); err != nil {
			return
		}
		cp = append(cp, p)
	}

	if len(cp) == 0 {
		return sql.ErrNoRows
	}

	*ps = cp

	return rows.Err()
}

/*
// The requests.
type (
	CreateProductRequest struct { // all required.
		CategoryID  int64  `json:"categoryID"`
		Title       string `json:"title"`
		ImageURL    string `json:"imageURL"`
		Price       float32 `json:"price"`
		Description string `json:"description"`
	}

	UpdateProductRequest CreateProductRequest // at least 1 required.

	GetProductRequest struct {
		ID int64 `json:"id"` // required.
	}

	DeleteProductRequest GetProductRequest

	GetProductsRequest struct {
		// [page, offset...]
	}
)
*/
