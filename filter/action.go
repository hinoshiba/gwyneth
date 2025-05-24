package filter

import (
	"os/exec"
	"fmt"
	"bufio"
	"strings"
	"syscall"
	"encoding/json"
)

import (
	"github.com/l4go/task"
)

import (
	"github.com/hinoshiba/gwyneth/slog"
	"github.com/hinoshiba/gwyneth/model"
	"github.com/hinoshiba/gwyneth/model/external"
)

type Action struct {
	id   *model.Id
	name string
	cmd  string
}

func NewAction(id *model.Id, name string, cmd string) *Action {
	return &Action{
		id: id,
		name: name,
		cmd: cmd,
	}
}

func (self *Action) Id() *model.Id {
	return self.id
}

func (self *Action) Name() string {
	return self.name
}

func (self *Action) Command() string {
	return self.cmd
}

func (self *Action) ConvertExternal() *external.Action {
	return &external.Action{
		Id: self.id.String(),
		Name: self.name,
		Cmd: self.cmd,
	}
}

func (self *Action) Do(msn *task.Mission, artcl *model.Article) error {
	defer msn.Done()
	slog.Debug("call '%s' '%s'", self.name, self.cmd)

	args := strings.SplitN(self.cmd, " ", 30)
	c := args[0]
	opts := []string{}
	if len(args) > 1 {
		opts = args[1:]
	}
	cmd := exec.CommandContext(msn.AsContext(), c, opts...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	output := ""
	stdout_scanner := bufio.NewScanner(stdout)
	go func(msn *task.Mission) {
		defer msn.Done()

		for stdout_scanner.Scan() {
			if task.IsCanceled(msn) {
				return
			}
			output += fmt.Sprintf("%s\n", stdout_scanner.Text())
		}
	}(msn.New())

	errout := ""
	stderr_scanner := bufio.NewScanner(stderr)
	go func(msn *task.Mission) {
		defer msn.Done()

		for stderr_scanner.Scan() {
			if task.IsCanceled(msn) {
				return
			}
			errout += fmt.Sprintf("%s\n", stderr_scanner.Text())
		}
	}(msn.New())

	if err := cmd.Start(); err != nil {
		msn.Cancel()
		return err
	}

	ext_artcle := artcl.ConvertExternal()
	stdin_val, err := json.Marshal(ext_artcle)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(stdin, string(stdin_val))
	if err != nil {
		return err
	}
	stdin.Close()

	if err := cmd.Wait(); err != nil {
		if cmd.Process != nil {
			cmd.Process.Signal(syscall.SIGKILL)
		}

		return fmt.Errorf("%s: %s", err, errout)
	}
	slog.Debug("%s successed: %s, %s", self.cmd, output, errout)
	return nil
}
