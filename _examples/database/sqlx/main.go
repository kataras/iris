package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/x/errors"
	"github.com/kataras/iris/v12/x/sqlx"

	_ "github.com/lib/pq"
)

const (
	host     = "localhost"
	port     = 5432
	user     = "postgres"
	password = "admin!123"
	dbname   = "test"
)

func main() {
	app := iris.New()

	db := mustConnectDB()
	mustCreateExtensions(context.Background(), db)
	mustCreateTables(context.Background(), db)

	app.Post("/", insert(db))
	app.Get("/", list(db))
	app.Get("/{event_id:uuid}", getByID(db))

	/*
		curl --location --request POST 'http://localhost:8080' \
		--header 'Content-Type: application/json' \
		--data-raw '{
		    "name": "second_test_event",
		    "data": {
		        "key": "value",
				"year": 2022
		    }
		}'

		curl --location --request GET 'http://localhost:8080'

		curl --location --request GET 'http://localhost:8080/4fc0363f-1d1f-4a43-8608-5ed266485645'
	*/
	app.Listen(":8080")
}

func mustConnectDB() *sql.DB {
	connString := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	db, err := sql.Open("postgres", connString)
	if err != nil {
		panic(err)
	}

	err = db.Ping()
	if err != nil {
		panic(err)
	}

	return db
}

func mustCreateExtensions(ctx context.Context, db *sql.DB) {
	query := `CREATE EXTENSION IF NOT EXISTS pgcrypto;`
	_, err := db.ExecContext(ctx, query)
	if err != nil {
		panic(err)
	}
}

func mustCreateTables(ctx context.Context, db *sql.DB) {
	query := `CREATE TABLE IF NOT EXISTS "events" (
		"id" uuid PRIMARY KEY NOT NULL DEFAULT gen_random_uuid(),
		"created_at" timestamp(6) DEFAULT now(),
		"name" text COLLATE "pg_catalog"."default",
		"data" jsonb
	  );`

	_, err := db.ExecContext(ctx, query)
	if err != nil {
		panic(err)
	}

	sqlx.Register("events", Event{})
}

type Event struct {
	ID        string          `json:"id"`
	CreatedAt time.Time       `json:"created_at"`
	Name      string          `json:"name"`
	Data      json.RawMessage `json:"data"`

	Presenter string `db:"-" json:"-"`
}

func insertEvent(ctx context.Context, db *sql.DB, evt Event) (id string, err error) {
	query := `INSERT INTO events(name,data) VALUES($1,$2) RETURNING id;`
	err = db.QueryRowContext(ctx, query, evt.Name, evt.Data).Scan(&id)
	return
}

func listEvents(ctx context.Context, db *sql.DB) ([]Event, error) {
	list := make([]Event, 0)
	query := `SELECT * FROM events ORDER BY created_at;`
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	// Not required. See sqlx.DefaultSchema.AutoCloseRows field.
	// defer rows.Close()

	if err = sqlx.Bind(&list, rows); err != nil {
		return nil, err
	}

	return list, nil
}

func getEvent(ctx context.Context, db *sql.DB, id string) (Event, error) {
	query := `SELECT * FROM events WHERE id = $1 LIMIT 1;`
	rows, err := db.QueryContext(ctx, query, id)
	if err != nil {
		return Event{}, err
	}

	var evt Event
	err = sqlx.Bind(&evt, rows)

	return evt, err
}

func insert(db *sql.DB) iris.Handler {
	return func(ctx iris.Context) {
		var evt Event
		if err := ctx.ReadJSON(&evt); err != nil {
			errors.InvalidArgument.Details(ctx, "unable to read body", err.Error())
			return
		}

		id, err := insertEvent(ctx, db, evt)
		if err != nil {
			errors.Internal.LogErr(ctx, err)
			return
		}

		ctx.JSON(iris.Map{"id": id})
	}
}

func list(db *sql.DB) iris.Handler {
	return func(ctx iris.Context) {
		events, err := listEvents(ctx, db)
		if err != nil {
			errors.Internal.LogErr(ctx, err)
			return
		}

		ctx.JSON(events)
	}
}

func getByID(db *sql.DB) iris.Handler {
	return func(ctx iris.Context) {
		eventID := ctx.Params().Get("event_id")

		evt, err := getEvent(ctx, db, eventID)
		if err != nil {
			errors.Internal.LogErr(ctx, err)
			return
		}

		ctx.JSON(evt)
	}
}
