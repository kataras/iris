// file: models/movie.go

package models

// Movie is our sample data structure.
type Movie struct {
	Name   string `json:"name"`
	Year   int    `json:"year"`
	Genre  string `json:"genre"`
	Poster string `json:"poster"`
}
