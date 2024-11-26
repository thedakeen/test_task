package data

import (
	"database/sql"
	"errors"
)

var (
	ErrNoRecordFound = errors.New("no records found")
	ErrEditConflict  = errors.New("edit conflict")
	ErrAlreadyExists = errors.New("song of this group is already exists")
)

type Models struct {
	Songs SongModel
}

func NewModels(db *sql.DB) Models {
	return Models{
		Songs: SongModel{DB: db},
	}
}
