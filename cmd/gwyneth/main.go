package main

import (
	"os"
	"fmt"
	"log/slog"
	"flag"
	"time"
)

import (
	"github.com/l4go/task"
)

import (
	"github.com/hinoshiba/gwyneth"
	"github.com/hinoshiba/gwyneth/http"
	"github.com/hinoshiba/gwyneth/config"
)

const (
	LOG_LEVEL slog.Level = slog.LevelDebug
)

var (
	Config *config.Config
)

func gwyneth_cmd() error {
	msn := task.NewMission()
	defer msn.Done()

	slog.Info("gwyneth starting...")

	g, err := gwyneth.New(msn.New(), Config)
	if err != nil {
		return err
	}
	defer g.Close()

	rt, err := http.New(msn.New(), Config.Http, g)
	if err != nil {
		return err
	}
	defer rt.Close()

	slog.Info("gwyneth started")

	if err := g.Test(); err != nil {
		return err
	}

	time.Sleep(time.Second * 60)
	slog.Info("gwyneth ending...")
	return nil
}

func init() {
	var c_path string
	flag.StringVar(&c_path, "c", "./gwyneth.cfg", "config path.")
	flag.Parse()

	if c_path == "" {
		die("config's path is empty")
	}
	cfg, err := config.Load(c_path)
	if err != nil {
		die("load error: %s", err)
	}
	Config = cfg

	slog.SetDefault(
		slog.New(
			slog.NewTextHandler(
				os.Stdout,
				&slog.HandlerOptions{Level: LOG_LEVEL},
			),
		),
	)
}

func die(s string, msg ...any) {
	fmt.Fprintf(os.Stderr, s + "\n", msg...)
	os.Exit(1)
}

func main() {
	if err := gwyneth_cmd(); err != nil {
		die("%s", err)
	}
}
