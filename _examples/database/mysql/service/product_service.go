package service

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"myapp/entity"
	"myapp/sql"
)

// ProductService represents the product entity service.
// Note that the given entity (request) should be already validated
// before service's calls.
type ProductService struct {
	*sql.Service
	rec sql.Record
}

// NewProductService returns a new product service to communicate with the database.
func NewProductService(db sql.Database) *ProductService {
	return &ProductService{Service: sql.NewService(db, new(entity.Product))}
}

// Insert stores a product to the database and returns its ID.
func (s *ProductService) Insert(ctx context.Context, e entity.Product) (int64, error) {
	if !e.ValidateInsert() {
		return 0, sql.ErrUnprocessable
	}

	q := fmt.Sprintf(`INSERT INTO %s (category_id, title, image_url, price, description)
	VALUES (?,?,?,?,?);`, e.TableName())

	res, err := s.DB().Exec(ctx, q, e.CategoryID, e.Title, e.ImageURL, e.Price, e.Description)
	if err != nil {
		return 0, err
	}

	return res.LastInsertId()
}

// BatchInsert inserts one or more products at once and returns the total length created.
func (s *ProductService) BatchInsert(ctx context.Context, products []entity.Product) (int, error) {
	if len(products) == 0 {
		return 0, nil
	}

	var (
		valuesLines []string
		args        []interface{}
	)

	for _, p := range products {
		if !p.ValidateInsert() {
			// all products should be "valid", we don't skip, we cancel.
			return 0, sql.ErrUnprocessable
		}

		valuesLines = append(valuesLines, "(?,?,?,?,?)")
		args = append(args, []interface{}{p.CategoryID, p.Title, p.ImageURL, p.Price, p.Description}...)
	}

	q := fmt.Sprintf("INSERT INTO %s (category_id, title, image_url, price, description) VALUES %s;",
		s.RecordInfo().TableName(),
		strings.Join(valuesLines, ", "))

	res, err := s.DB().Exec(ctx, q, args...)
	if err != nil {
		return 0, err
	}

	n := sql.GetAffectedRows(res)
	return n, nil
}

// Update updates a product based on its `ID` from the database
// and returns the affected numbrer (0 when nothing changed otherwise 1).
func (s *ProductService) Update(ctx context.Context, e entity.Product) (int, error) {
	q := fmt.Sprintf(`UPDATE %s
    SET
	    category_id = ?,
	    title = ?,
	    image_url = ?,
	    price = ?,
	    description = ?
	WHERE %s = ?;`, e.TableName(), e.PrimaryKey())

	res, err := s.DB().Exec(ctx, q, e.CategoryID, e.Title, e.ImageURL, e.Price, e.Description, e.ID)
	if err != nil {
		return 0, err
	}

	n := sql.GetAffectedRows(res)
	return n, nil
}

var productUpdateSchema = map[string]reflect.Kind{
	"category_id": reflect.Int,
	"title":       reflect.String,
	"image_url":   reflect.String,
	"price":       reflect.Float32,
	"description": reflect.String,
}

// PartialUpdate accepts a key-value map to
// update the record based on the given "id".
func (s *ProductService) PartialUpdate(ctx context.Context, id int64, attrs map[string]interface{}) (int, error) {
	return s.Service.PartialUpdate(ctx, id, productUpdateSchema, attrs)
}
