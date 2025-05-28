package main

import (
	"os"
	"os/signal"
	"fmt"
	"flag"
	"syscall"
)

import (
	"github.com/l4go/task"
)

import (
	"github.com/hinoshiba/gwyneth"
	"github.com/hinoshiba/gwyneth/slog"
	"github.com/hinoshiba/gwyneth/http"
	"github.com/hinoshiba/gwyneth/config"
)

var (
	Config *config.Config
)

func gwyneth_cmd() error {
	msn := task.NewMission()

	lm, err := slog.New(msn.New(), Config.Log)
	if err != nil {
		return err
	}
	defer lm.Close()

	slog.Info("gwyneth starting...")
	defer slog.Info("gwyneth ending...")

	go watch_sighup(msn.New(), lm)

	g, err := gwyneth.New(msn.New(), lm, Config)
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
}

func watch_sighup(msn *task.Mission, lm *slog.LogManager) {
	defer msn.Done()

	sig_ch := make(chan os.Signal, 1)
	signal.Notify(sig_ch, syscall.SIGHUP)

	select {
	case <- msn.RecvCancel():
		return
	case <- sig_ch:
		lm.Restart()
	}
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
