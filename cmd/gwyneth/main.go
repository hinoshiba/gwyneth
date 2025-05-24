package main

import (
	"os"
	"fmt"
	"log/slog"
	"flag"
)

import (
	"github.com/l4go/task"
)

import (
	"github.com/hinoshiba/gwyneth"
	"github.com/hinoshiba/gwyneth/http"
	"github.com/hinoshiba/gwyneth/config"
)

var (
	Config *config.Config
)

func gwyneth_cmd() error {
	msn := task.NewMission()
	slog.Info("gwyneth starting...")
	defer slog.Info("gwyneth ending...")

	g, err := gwyneth.New(msn.New(), Config)
	if err != nil {
		return err
	}
	defer g.Close()

	rt, err := http.New(msn.New(), Config, g)
	if err != nil {
		return err
	}
	defer rt.Close()

	slog.Info("gwyneth started")
	msn.Done()

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
				&slog.HandlerOptions{Level: Config.Log.GetSlogLevel()},
			),
		),
	)
	slog.Info(fmt.Sprintf("LogLevel: %s", cfg.Log.Level))
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
