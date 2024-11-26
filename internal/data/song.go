package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/lib/pq"
	"test_task/internal/validator"
	"time"
)

// @Description Song data structure
// @Schema
type Song struct {
	ID        int64     `json:"id"`
	Song      string    `json:"song"`
	Group     string    `json:"group"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`

	Release string `json:"releaseDate"`
	Text    string `json:"text"`
	Link    string `json:"link"`
}

type SongModel struct {
	DB *sql.DB
}

func ValidateSong(v *validator.Validator, song *Song) {
	v.Check(song.Song != "", "song", "must be provided")
	v.Check(song.Group != "", "group", "must be provided")
}

func (s SongModel) Insert(song *Song) error {
	query := `
INSERT INTO songs (song_name, group_name, release, text, link)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, created_at, updated_at`

	args := []any{song.Song, song.Group, song.Release, song.Text, song.Link}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := s.DB.QueryRowContext(ctx, query, args...).Scan(&song.ID, &song.CreatedAt, &song.UpdatedAt)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return ErrAlreadyExists
		}
		return err
	}

	return nil
}

func (s SongModel) Update(song *Song) error {
	query := `
UPDATE songs
SET song_name = $1, group_name = $2,updated_at = NOW()
WHERE id = $3
RETURNING updated_at`

	args := []any{
		song.Song,
		song.Group,
		song.ID,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := s.DB.QueryRowContext(ctx, query, args...).Scan(
		&song.UpdatedAt)

	if err != nil {
		var pqErr *pq.Error
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrEditConflict
		case errors.As(err, &pqErr) && pqErr.Code == "23505":
			return ErrAlreadyExists
		default:
			return err
		}
	}
	return nil
}

func (s SongModel) GetAll(song, group, release, text, link string, filters Filters) ([]*Song, Metadata, error) {
	query := fmt.Sprintf(`
SELECT count(*) OVER(), id, created_at, updated_at, song_name, group_name, release, text, link
FROM songs
WHERE (to_tsvector('simple', song_name) @@ plainto_tsquery('simple', $1) OR $1 = '')
AND (to_tsvector('simple', group_name) @@ plainto_tsquery('simple', $2) OR $2 = '')
AND (to_tsvector('simple', release) @@ plainto_tsquery('simple', $3) OR $3 = '')
AND (to_tsvector('simple', text) @@ plainto_tsquery('simple', $4) OR $4 = '')
AND (to_tsvector('simple', link) @@ plainto_tsquery('simple', $5) OR $5 = '')
ORDER BY %s %s, id ASC
LIMIT $6 OFFSET $7`,
		filters.sortColumn(), filters.sortDirection())

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	args := []any{song, group, release, text, link, filters.limit(), filters.offset()}

	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err
	}

	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
		}
	}(rows)

	totalRecords := 0
	var songs []*Song

	for rows.Next() {
		song := &Song{}

		err := rows.Scan(
			&totalRecords,
			&song.ID,
			&song.CreatedAt,
			&song.UpdatedAt,
			&song.Song,
			&song.Group,
			&song.Release,
			&song.Text,
			&song.Link)

		if err != nil {
			return nil, Metadata{}, err
		}

		songs = append(songs, song)
	}

	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}

	metadata := calculateMetaData(totalRecords, filters.Page, filters.PageSize)

	return songs, metadata, nil

}

//////////////////////////////////////////////////////////

func (s SongModel) Get(id int64) (*Song, error) {
	if id < 1 {
		return nil, ErrNoRecordFound
	}

	query := `
SELECT id, created_at, updated_at, song_name, group_name, release, text, link 
FROM songs
WHERE id = $1`

	var song Song

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := s.DB.QueryRowContext(ctx, query, id).Scan(
		&song.ID,
		&song.CreatedAt,
		&song.UpdatedAt,
		&song.Song,
		&song.Group,
		&song.Release,
		&song.Text,
		&song.Link)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrNoRecordFound
		default:
			return nil, err
		}
	}

	return &song, nil
}

func (s SongModel) GetLyrics(id int64, page int, pageSize int) ([]string, Metadata, error) {
	if id < 1 {
		return nil, Metadata{}, ErrNoRecordFound
	}

	query := `
SELECT text 
FROM songs
WHERE id = $1`

	var song Song

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := s.DB.QueryRowContext(ctx, query, id).Scan(
		&song.Text)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, Metadata{}, ErrNoRecordFound
		default:
			return nil, Metadata{}, err
		}
	}

	verses := splitTextIntoVerses(song.Text)

	totalRecords := len(verses)
	start := (page - 1) * pageSize
	end := start + pageSize
	if end > totalRecords {
		end = totalRecords
	}

	metadata := calculateMetaData(totalRecords, page, pageSize)

	return verses[start:end], metadata, nil
}

func (s *SongModel) Delete(id int64) error {
	if id < 1 {
		return ErrNoRecordFound
	}

	query := `
DELETE FROM songs
WHERE id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := s.DB.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrNoRecordFound
	}

	return nil
}
