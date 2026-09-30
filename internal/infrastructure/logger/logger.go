package logger

import (
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"
)

type Logger struct {
	zerolog.Logger
}

func New() Logger {
	logDir := "logs"
	_ = os.MkdirAll(logDir, 0755)

	fileWriter := &lumberjack.Logger{
		Filename:   filepath.Join(logDir, "app.log"), // Путь, где будут лежать сохраненные логи
		MaxSize:    10,                               // Как только файл станет 10 МБ — сжимаем его в архив
		MaxBackups: 3,                                // Храним максимум 3 старых архива на диске
		Compress:   true,                             // Архивируем старые логи в .gz, чтобы экономить место
	}

	consoleWriter := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
	}

	mw := io.MultiWriter(consoleWriter, fileWriter)
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	internalZl := zerolog.New(mw).With().Timestamp().Logger()

	return Logger{internalZl}
}
