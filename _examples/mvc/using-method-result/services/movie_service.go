// file: services/movie_service.go

package services

import (
	"errors"
	"sync"

	"github.com/kataras/iris/_examples/mvc/using-method-result/models"
)

// MovieService handles CRUID operations of a movie entity/model.
// It's here to decouple the data source from the higher level compoments.
// As a result a different service for a specific datasource (or repository)
// can be used from the main application without any additional changes.
type MovieService interface {
	GetSingle(query func(models.Movie) bool) (movie models.Movie, found bool)
	GetByID(id int64) (models.Movie, bool)

	InsertOrUpdate(movie models.Movie) (models.Movie, error)
	DeleteByID(id int64) bool

	GetMany(query func(models.Movie) bool, limit int) (result []models.Movie)
	GetAll() []models.Movie
}

// NewMovieServiceFromMemory returns a new memory-based movie service.
func NewMovieServiceFromMemory(source map[int64]models.Movie) MovieService {
	return &MovieMemoryService{
		source: source,
	}
}

// A Movie Service can have different data sources:
// func NewMovieServiceFromDB(db datasource.MySQL) {
// 	return &MovieDatabaseService{
// 		db: db,
// 	}
// }

// Another pattern is to initialize the database connection
// or any source here based on a "string" name or an "enum".
// func NewMovieService(source string) MovieService {
// 	if source == "memory" {
// 		return NewMovieServiceFromMemory(datasource.Movies)
// 	}
//	if source == "database" {
//		db = datasource.NewDB("....")
//		return NewMovieServiceFromDB(db)
//	}
// 	[...]
// 	return nil
// }

// MovieMemoryService is a "MovieService"
// which manages the movies using the memory data source (map).
type MovieMemoryService struct {
	source map[int64]models.Movie
	mu     sync.RWMutex
}

// GetSingle receives a query function
// which is fired for every single movie model inside
// our imaginary data source.
// When that function returns true then it stops the iteration.
//
// It returns the query's return last known boolean value
// and the last known movie model
// to help callers to reduce the LOC.
//
// It's actually a simple but very clever prototype function
// I'm using everywhere since I firstly think of it,
// hope you'll find it very useful as well.
func (s *MovieMemoryService) GetSingle(query func(models.Movie) bool) (movie models.Movie, found bool) {
	s.mu.RLock()
	for _, movie = range s.source {
		found = query(movie)
		if found {
			break
		}
	}
	s.mu.RUnlock()

	// set an empty models.Movie if not found at all.
	if !found {
		movie = models.Movie{}
	}

	return
}

// GetByID returns a movie based on its id.
// Returns true if found, otherwise false, the bool should be always checked
// because the models.Movie may be filled with the latest element
// but not the correct one, although it can be used for debugging.
func (s *MovieMemoryService) GetByID(id int64) (models.Movie, bool) {
	return s.GetSingle(func(m models.Movie) bool {
		return m.ID == id
	})
}

// InsertOrUpdate adds or updates a movie to the (memory) storage.
//
// Returns the new movie and an error if any.
func (s *MovieMemoryService) InsertOrUpdate(movie models.Movie) (models.Movie, error) {
	id := movie.ID

	if id == 0 { // Create new action
		var lastID int64
		// find the biggest ID in order to not have duplications
		// in productions apps you can use a third-party
		// library to generate a UUID as string.
		s.mu.RLock()
		for _, item := range s.source {
			if item.ID > lastID {
				lastID = item.ID
			}
		}
		s.mu.RUnlock()

		id = lastID + 1
		movie.ID = id

		// map-specific thing
		s.mu.Lock()
		s.source[id] = movie
		s.mu.Unlock()

		return movie, nil
	}

	// Update action based on the movie.ID,
	// here we will allow updating the poster and genre if not empty.
	// Alternatively we could do pure replace instead:
	// s.source[id] = movie
	// and comment the code below;
	current, exists := s.GetByID(id)
	if !exists { // ID is not a real one, return an error.
		return models.Movie{}, errors.New("failed to update a nonexistent movie")
	}

	// or comment these and s.source[id] = m for pure replace
	if movie.Poster != "" {
		current.Poster = movie.Poster
	}

	if movie.Genre != "" {
		current.Genre = movie.Genre
	}

	// map-specific thing
	s.mu.Lock()
	s.source[id] = current
	s.mu.Unlock()

	return movie, nil
}

// DeleteByID deletes a movie by its id.
//
// Returns true if deleted otherwise false.
func (s *MovieMemoryService) DeleteByID(id int64) bool {
	if _, exists := s.GetByID(id); !exists {
		// we could do _, exists := s.source[id] instead
		// but we don't because you should learn
		// how you can use that service's functions
		// with any other source, i.e database.
		return false
	}

	// map-specific thing
	s.mu.Lock()
	delete(s.source, id)
	s.mu.Unlock()

	return true
}

// GetMany same as GetSingle but returns one or more models.Movie as a slice.
// If limit <=0 then it returns everything.
func (s *MovieMemoryService) GetMany(query func(models.Movie) bool, limit int) (result []models.Movie) {
	loops := 0

	s.mu.RLock()
	for _, movie := range s.source {
		loops++

		passed := query(movie)
		if passed {
			result = append(result, movie)
		}
		// we have to return at least one movie if "passed" was true.
		if limit >= loops {
			break
		}
	}
	s.mu.RUnlock()

	return
}

// GetAll returns all movies.
func (s *MovieMemoryService) GetAll() []models.Movie {
	movies := s.GetMany(func(m models.Movie) bool { return true }, -1)
	return movies
}
