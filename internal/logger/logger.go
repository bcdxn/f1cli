package logger

import (
	"log"
	"log/slog"
	"os"
)

func New() (*slog.Logger, *os.File) {
	file, err := os.Create("app.log")
	if err != nil {
		log.Fatal(err)
	}

	// Create a text handler that writes to the file
	handler := slog.NewTextHandler(file, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	// Create a logger with the file handler
	return slog.New(handler), file
}
