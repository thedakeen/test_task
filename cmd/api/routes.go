package main

import (
	"github.com/julienschmidt/httprouter"
	"github.com/swaggo/http-swagger"
	"net/http"
)

func (app *application) routes() http.Handler {
	router := httprouter.New()

	router.NotFound = http.HandlerFunc(app.notFoundResponse)

	router.HandlerFunc(http.MethodGet, "/v1/healthcheck", app.healthcheckHandler)

	router.HandlerFunc(http.MethodPost, "/v1/song", app.addSongHandler)

	router.HandlerFunc(http.MethodGet, "/v1/song/:id", app.showSongHandler)
	router.HandlerFunc(http.MethodPatch, "/v1/song/:id", app.updateSongHandler)
	router.HandlerFunc(http.MethodDelete, "/v1/song/:id", app.deleteSongHandler)

	router.HandlerFunc(http.MethodGet, "/v1/song/:id/lyrics", app.showLyricsHandler)

	router.HandlerFunc(http.MethodGet, "/v1/songs", app.listSongsHandler)

	router.Handler(http.MethodGet, "/swagger/*filepath", httpSwagger.WrapHandler)

	return router
}
