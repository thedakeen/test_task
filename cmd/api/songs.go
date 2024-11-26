package main

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"test_task/internal/data"
	"test_task/internal/validator"

	_ "test_task/cmd/api/docs"
)

// @Summary Add a new song
// @Description Adds a new song with its details fetched from an external API
// @Tags Songs
// @Accept json
// @Produce json
// @Param song body object{song=string,group=string} true "Song and Group"
// @Success 201 {object} map[string]string "Song added successfully"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 422 {object} map[string]string "Validation errors"
// @Failure 500 {object} map[string]string "the server encountered a problem and could not process your request"
// @Router /v1/song [post]
func (app *application) addSongHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Song  string `json:"song"`
		Group string `json:"group"`
	}

	log := app.logger.With(
		slog.String("group", req.Group),
		slog.String("song", req.Song))

	log.Info("attempting to add a new song")

	err := app.readJSON(w, r, &req)
	if err != nil {
		app.badRequestResponse(w, r, err)
		app.logger.Warn("bad request", err)
		return
	}

	songDetail, err := app.fetchSongDetail(req.Song, req.Group)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		app.logger.Error("failed to fetch song details")
		return
	}

	song := &data.Song{
		Song:    req.Song,
		Group:   req.Group,
		Release: songDetail.Release,
		Text:    songDetail.Text,
		Link:    songDetail.Link,
	}

	v := validator.New()

	if data.ValidateSong(v, song); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		app.logger.Warn("validation has not passed", err)
		return
	}

	err = app.models.Songs.Insert(song)
	if err != nil {
		if errors.Is(err, data.ErrAlreadyExists) {
			v.AddError("song", "a song of this group is already exists")
			app.failedValidationResponse(w, r, v.Errors)
			app.logger.Warn("song is already in database", err)
			return
		}
	}

	err = app.writeJSON(w, http.StatusCreated, envelope{"song": song}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		app.logger.Error("failed to add song")
		return
	}

	log.Info("song added successfully")
}

// @Summary Update a song
// @Description Updates an existing song's details by its ID
// @Tags Songs
// @Accept json
// @Produce json
// @Param id path int true "Song ID"
// @Param song body object{song=string,group=string} false "Updated song and group (optional)"
// @Success 200 {object} map[string]string "Song updated successfully"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 404 {object} map[string]string "Song not found"
// @Failure 409 {object} map[string]string "Edit conflict or duplicate song"
// @Failure 422 {object} map[string]string "Validation errors"
// @Failure 500 {object} map[string]string "the server encountered a problem and could not process your request"
// @Router /v1/song/{id} [patch]
func (app *application) updateSongHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	log := app.logger.With(
		slog.String("song", strconv.FormatInt(id, 10)))

	log.Info("attempting to edit a song")

	song, err := app.models.Songs.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrNoRecordFound):
			app.notFoundResponse(w, r)
			app.logger.Warn("song not found", err)
		default:
			app.serverErrorResponse(w, r, err)
			app.logger.Error("error updating song", err)
		}
		return
	}

	var req struct {
		Song  *string `json:"song"`
		Group *string `json:"group"`
	}

	err = app.readJSON(w, r, &req)
	if err != nil {
		app.badRequestResponse(w, r, err)
		log.Error("bad request", err)
		return
	}

	if req.Song != nil {
		song.Song = *req.Song
	}

	if req.Group != nil {
		song.Group = *req.Group
	}

	v := validator.New()

	if data.ValidateSong(v, song); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		app.logger.Warn("validation has not passed", err)
		return
	}

	err = app.models.Songs.Update(song)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w, r)
			app.logger.Warn("edit conflict error", err)
		case errors.Is(err, data.ErrAlreadyExists):
			v.AddError("song", "a song of this group is already exists")
			app.failedValidationResponse(w, r, v.Errors)
			app.logger.Warn("song is already in database", err)
		default:
			app.serverErrorResponse(w, r, err)
			app.logger.Error("error updating song", err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"song": song}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		app.logger.Error("error updating song", err)
	}

	log.Info("song was edited successfully")

}

// @Summary Delete a song
// @Description Deletes a song by its ID
// @Tags Songs
// @Accept json
// @Produce json
// @Param id path int true "Song ID"
// @Success 200 {object} map[string]string "Song successfully deleted"
// @Failure 404 {object} map[string]string "Song not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /v1/song/{id} [delete]
func (app *application) deleteSongHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	log := app.logger.With(
		slog.String("song", strconv.FormatInt(id, 10)))

	log.Info("trying to delete the song")

	err = app.models.Songs.Delete(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrNoRecordFound):
			app.notFoundResponse(w, r)
			app.logger.Warn("song not found", err)
		default:
			app.serverErrorResponse(w, r, err)
			app.logger.Error("error deleting song", err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"message": "song successfully deleted"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		app.logger.Error("error deleting song", err)
	}

	log.Info("song successfully deleted")
}

// @Summary List songs with filters
// @Description Lists all songs with optional filters and pagination
// @Tags Songs
// @Accept json
// @Produce json
// @Param song query string false "Song name"
// @Param group query string false "Group name"
// @Param releaseDate query string false "Release date"
// @Param text query string false "Text"
// @Param link query string false "Link"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Number of items per page" default(5)
// @Param sort query string false "Sort by field" default(id) Enum(id,song,group,release,text,link,-id,-song,-group,-release,-text,-link)
// @Success 200 {object} envelope{songs=[]data.Song,metadata=data.Metadata} "List of songs"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 500 {object} map[string]string "the server encountered a problem and could not process your request"
// @Router /v1/songs [get]
func (app *application) listSongsHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Song    string `json:"song"`
		Group   string `json:"group"`
		Release string `json:"releaseDate"`
		Text    string `json:"text"`
		Link    string `json:"link"`
		data.Filters
	}

	v := validator.New()

	qs := r.URL.Query()

	req.Song = app.readString(qs, "song", "")
	req.Group = app.readString(qs, "group", "")
	req.Release = app.readString(qs, "releaseDate", "")
	req.Text = app.readString(qs, "text", "")
	req.Link = app.readString(qs, "link", "")

	req.Filters.Page = app.readInt(qs, "page", 1, v)
	req.Filters.PageSize = app.readInt(qs, "page_size", 5, v)

	req.Filters.Sort = app.readString(qs, "sort", "id")

	req.Filters.SortSafelist = []string{"id", "song", "group", "release", "text", "link", "-id", "-song", "-group", "-release", "-text", "-link"}

	if data.ValidateFilters(v, req.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		app.logger.Warn("validation has not passed")
		return
	}

	log := app.logger.With(
		slog.String("song filter", req.Song),
		slog.String("group filter", req.Group),
		slog.Int("page filter", req.Page))

	log.Info("trying to list songs")

	songs, metadata, err := app.models.Songs.GetAll(req.Song, req.Group, req.Release, req.Text, req.Link, req.Filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		app.logger.Error("error getting songs", err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"songs": songs, "metadata": metadata}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		app.logger.Error("error getting songs", err)
	}

	log.Info("listed successfully")
}

// @Summary Get paginated lyrics of a song
// @Description Retrieves the lyrics of a song with pagination by verse
// @Tags Songs
// @Accept json
// @Produce json
// @Param id path int true "Song ID"
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Number of verses per page" default(1)
// @Success 200 {object} envelope{lyrics=[]string,metadata=data.Metadata} "Paginated song lyrics"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 404 {object} map[string]string "Song not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /v1/song/{id}/lyrics [get]
func (app *application) showLyricsHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		data.Filters
	}

	v := validator.New()
	qs := r.URL.Query()

	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	req.Filters.Page = app.readInt(qs, "page", 1, v)
	req.Filters.PageSize = app.readInt(qs, "page_size", 1, v)

	log := app.logger.With(
		slog.String("song", strconv.FormatInt(id, 10)))

	log.Info("attempting to get song's lyrics")

	lyrics, metadata, err := app.models.Songs.GetLyrics(id, req.Page, req.PageSize)

	if err != nil {
		switch {
		case errors.Is(err, data.ErrNoRecordFound):
			app.notFoundResponse(w, r)
			app.logger.Warn("song not found", err)
		default:
			app.serverErrorResponse(w, r, err)
			app.logger.Error("failed to get song", err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"lyrics": lyrics, "metadata": metadata}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		app.logger.Error("failed to get song's lyrics", err)
	}

	log.Info("song's lyrics was gotten successfully")
}

///////////////////////////////////

// @Summary Get a song by ID
// @Description Retrieves a song by its ID
// @Tags Songs
// @Accept json
// @Produce json
// @Param id path int true "Song ID"
// @Success 200 {object} envelope{song=data.Song} "Successfully retrieved song"
// @Failure 404 {object} map[string]string "The requested resource could not be found"
// @Failure 500 {object} map[string]string "The server encountered a problem and could not process your request"
// @Router /v1/song/{id} [get]
func (app *application) showSongHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	log := app.logger.With(
		slog.String("song", strconv.FormatInt(id, 10)))

	log.Info("attempting to get song")

	song, err := app.models.Songs.Get(id)

	if err != nil {
		switch {
		case errors.Is(err, data.ErrNoRecordFound):
			app.notFoundResponse(w, r)
			app.logger.Warn("song not found", err)
		default:
			app.serverErrorResponse(w, r, err)
			app.logger.Error("failed to get song", err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"song": song}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		app.logger.Error("failed to get song", err)
	}

	log.Info("song was gotten successfully")
}
