package slog

import (
	"os"
	"fmt"
	"log/slog"
	"sync"
	"path/filepath"
)

import (
	"github.com/l4go/task"
)

import (
	"github.com/hinoshiba/gwyneth/config"
)

const (
	FNAME_COLLECTOR = "collector.log"
	FNAME_ACTION = "action.log"
)

func Debug(s string, msg ...any) {
	val := fmt.Sprintf(s, msg...)
	slog.Debug(val)
}

func Info(s string, msg ...any) {
	val := fmt.Sprintf(s, msg...)
	slog.Info(val)
}

func Warn(s string, msg ...any) {
	val := fmt.Sprintf(s, msg...)
	slog.Warn(val)
}

func Error(s string, msg ...any) {
	val := fmt.Sprintf(s, msg...)
	slog.Error(val)
}

type Logger struct {
	id string

	lm *LogManager
}

func (l *Logger) Debug(s string, msg ...any) {
	l.lm.debug4id(l.id, s, msg...)
}

func (l *Logger) Info(s string, msg ...any) {
	l.lm.info4id(l.id, s, msg...)
}

func (l *Logger) Warn(s string, msg ...any) {
	l.lm.warn4id(l.id, s, msg...)
}

func (l *Logger) Error(s string, msg ...any) {
	l.lm.error4id(l.id, s, msg...)
}

type logger struct {
	f *os.File
	L *slog.Logger
}

func newLogger(path string, level slog.Level) (*logger, error){
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	return &logger{
		f: f,
		L: slog.New(slog.NewTextHandler(f, &slog.HandlerOptions{Level: level})),
	}, nil
}

func (l *logger) Close() error {
	return l.f.Close()
}

type LogManager struct {
	dir string
	cfg *config.Log

	loggers map[string]*logger

	mtx *sync.RWMutex
	msn *task.Mission
}

func New(msn *task.Mission, cfg *config.Log) (*LogManager, error) {
	dir := filepath.Clean(cfg.Dir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		msn.Done()
		return nil, err
	}

	lm := &LogManager{
		dir: dir,
		cfg: cfg,
		loggers: map[string]*logger{
			FNAME_COLLECTOR: nil,
			FNAME_ACTION: nil,
		},
		mtx: new(sync.RWMutex),
		msn: msn,
	}

	lm.Restart()
	return lm, nil
}

func (lm *LogManager) debug4id(id string, s string, msg ...any) {
	lm.mtx.RLock()
	defer lm.mtx.RUnlock()

	val := fmt.Sprintf(s, msg...)
	l := lm.loggers[id]
	l.L.Debug(val)
}

func (lm *LogManager) info4id(id string, s string, msg ...any) {
	lm.mtx.RLock()
	defer lm.mtx.RUnlock()

	val := fmt.Sprintf(s, msg...)
	l := lm.loggers[id]
	l.L.Info(val)
}

func (lm *LogManager) warn4id(id string, s string, msg ...any) {
	lm.mtx.RLock()
	defer lm.mtx.RUnlock()

	val := fmt.Sprintf(s, msg...)
	l := lm.loggers[id]
	l.L.Warn(val)
}

func (lm *LogManager) error4id(id string, s string, msg ...any) {
	lm.mtx.RLock()
	defer lm.mtx.RUnlock()

	val := fmt.Sprintf(s, msg...)
	l := lm.loggers[id]
	l.L.Error(val)
}

func (lm *LogManager) GetCollectorsLogger() *Logger {
	return &Logger{
		id: FNAME_COLLECTOR,
		lm: lm,
	}
}

func (lm *LogManager) GetActionsLogger() *Logger {
	return &Logger{
		id: FNAME_ACTION,
		lm: lm,
	}
}

func (lm *LogManager) Restart() error {
	lm.mtx.Lock()
	defer lm.mtx.Unlock()

	for k, _ := range lm.loggers {
		if logger := lm.loggers[k]; logger != nil {
			if err := logger.Close(); err != nil {
				return err
			}
		}

		path := filepath.Join(lm.dir, k)
		logger, err := newLogger(path, lm.cfg.GetSlogLevel())
		if err != nil {
			return err
		}
		lm.loggers[k] = logger
	}

	slog.SetDefault(
		slog.New(
			slog.NewTextHandler(
				os.Stdout,
				&slog.HandlerOptions{Level: lm.cfg.GetSlogLevel()},
			),
		),
	)

	Info("LogManager Reloaded: LogLevel: %s", lm.cfg.Level)
	return nil
}

func (lm *LogManager) Close() error {
	defer lm.msn.Done()

	lm.mtx.Lock()
	defer lm.mtx.Unlock()

	lm.msn.Cancel()
	return nil
}
