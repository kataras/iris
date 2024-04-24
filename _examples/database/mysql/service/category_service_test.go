package service

import (
	"context"
	"testing"

	"myapp/entity"
	"myapp/sql"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestCategoryServiceInsert(t *testing.T) {
	conn, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	db := &sql.MySQL{Conn: conn}
	service := NewCategoryService(db)
	newCategory := entity.Category{
		Title:    "computer-internet",
		Position: 2,
		ImageURL: "https://animage",
	}
	mock.ExpectExec("INSERT INTO categories (title, position, image_url) VALUES (?,?,?);").
		WithArgs(newCategory.Title, newCategory.Position, newCategory.ImageURL).WillReturnResult(sqlmock.NewResult(1, 1))

	id, err := service.Insert(context.TODO(), newCategory)
	if err != nil {
		t.Fatal(err)
	}

	if id != 1 {
		t.Fatalf("expected ID to be 1 as this is the first entry")
	}

	if err = mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
