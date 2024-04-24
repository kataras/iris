package sqlx

/*
import (
	"reflect"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

type food struct {
	ID        string
	Name      string
	Presenter bool `db:"-"`
}

func TestTableBind(t *testing.T) {
	Register("foods", food{})

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}

	mock.ExpectQuery("SELECT .* FROM foods WHERE id = ?").
		WithArgs("42").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).
			AddRow("42", "banana").
			AddRow("43", "broccoli"))

	rows, err := db.Query("SELECT .* FROM foods WHERE id = ? LIMIT 1", "42")
	if err != nil {
		t.Fatal(err)
	}

	var f food
	err = Bind(&f, rows)
	if err != nil {
		t.Fatal(err)
	}

	expectedSingle := food{"42", "banana", false}
	if !reflect.DeepEqual(f, expectedSingle) {
		t.Fatalf("expected value: %#+v but got: %#+v", expectedSingle, f)
	}

	mock.ExpectQuery("SELECT .* FROM foods").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).
			AddRow("42", "banana").
			AddRow("43", "broccoli").
			AddRow("44", "chicken"))
	rows, err = db.Query("SELECT .* FROM foods")
	if err != nil {
		t.Fatal(err)
	}

	var foods []food
	err = Bind(&foods, rows)
	if err != nil {
		t.Fatal(err)
	}

	expectedMany := []food{
		{"42", "banana", false},
		{"43", "broccoli", false},
		{"44", "chicken", false},
	}

	for i := range foods {
		if !reflect.DeepEqual(foods[i], expectedMany[i]) {
			t.Fatalf("[%d] expected: %#+v but got: %#+v", i, expectedMany[i], foods[i])
		}
	}
}
*/
