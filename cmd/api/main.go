package main

import (
	"context"
	"database/sql"
	"flag"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"log/slog"
	"os"
	"sync"
	"test_task/internal/data"
	"time"
)

const version = "1.0.0"

type config struct {
	port  int
	env   string
	dbDSN string
}

type application struct {
	config config
	logger *slog.Logger
	models data.Models
	wg     sync.WaitGroup
}

// @title Music Library
// @version 1.0.0
// @description API server for test task application

// @host localhost:5000
// @BasePath /

func main() {
	err := godotenv.Load()
	if err != nil {
		panic("no .env file found")
	}

	var cfg config
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	log.Info("database connection established")

	flag.IntVar(&cfg.port, "port", 5000, "API Server Port")
	flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")
	flag.StringVar(&cfg.dbDSN, "db-dsn", "", "PostgreSQL DSN")

	if cfg.dbDSN == "" {
		cfg.dbDSN = os.Getenv("DB_DSN")
	}

	db, err := openDB(cfg)
	if err != nil {
		log.Error("Fatal error occurred",
			"error", err.Error(),
			"level", "fatal")

		os.Exit(1)
	}

	defer db.Close()

	app := application{
		config: cfg,
		logger: log,
		models: data.NewModels(db),
	}

	err = app.serve()
	if err != nil {
		log.Error("Fatal error occurred",
			"error", err.Error(),
			"level", "fatal")
	}
}

func openDB(cfg config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.dbDSN)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		return nil, err
	}

	return db, nil
}
