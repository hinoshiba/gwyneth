package filter

import (
	"os"
	"fmt"
	"sync"
	"path/filepath"
	"encoding/json"
)

import (
	"github.com/l4go/task"
	"github.com/fsnotify/fsnotify"
)

import (
	"github.com/hinoshiba/gwyneth/model"
	"github.com/hinoshiba/gwyneth/model/external"
	"github.com/hinoshiba/gwyneth/slog"
	"github.com/hinoshiba/gwyneth/config"
)


type ActionManager struct {
	action    *Action

	path_q    string
	path_tmp  string
	path_wip  string
	path_dlq  string

	logger    *slog.Logger

	fpath_ch chan string

	session_mtx *sync.Mutex
	session_cc  task.Canceller

	msn *task.Mission
}

func NewActionManager(msn *task.Mission, action *Action, cfg *config.Action, logger *slog.Logger) (*ActionManager, error) {
	path_qbase := filepath.Join(cfg.QueueDir, action.Id().String())
	path_q := filepath.Join(path_qbase, "new")
	if err := os.MkdirAll(path_q, 0755); err != nil {
		return nil, err
	}
	path_tmp := filepath.Join(path_qbase, "tmp")
	if err := os.MkdirAll(path_tmp, 0755); err != nil {
		return nil, err
	}
	path_wip := filepath.Join(path_qbase, "wip")
	if err := os.MkdirAll(path_wip, 0755); err != nil {
		return nil, err
	}
	path_dlq := filepath.Join(path_qbase, "deadletter")
	if err := os.MkdirAll(path_dlq, 0755); err != nil {
		return nil, err
	}

	self := &ActionManager{
		action: action,

		path_q: path_q,
		path_tmp: path_tmp,
		path_wip: path_wip,
		path_dlq: path_dlq,

		logger: logger,

		fpath_ch: make(chan string),

		session_mtx: new(sync.Mutex),

		msn: msn,
	}
	self.run()
	return self, nil
}

func (self *ActionManager) run() {
	self.newSession()
	go self.task_handler(self.msn.New())

	self.run_f_watcher()
}

func (self *ActionManager) Close() {
	defer close(self.fpath_ch)
	defer self.msn.Done()

	self.msn.Cancel()
}

func (self *ActionManager) AddQueueItem(id *model.Id, body []byte) error {
	tmpfile, err := os.CreateTemp(self.path_tmp, "tmp-*")
	if err != nil {
		return err
	}
	if _, err := tmpfile.Write(body); err != nil {
		return err
	}
	tmpfile.Close()

	path := filepath.Join(self.path_q, id.String())
	slog.Debug("ActionManager.Write item %s -> %s", tmpfile.Name(), path)
	return os.Rename(tmpfile.Name(), path)
}

func (self *ActionManager) GetQueueItems() ([]*model.Article, error) {
	return getQueueItems(self.path_q)
}

func (self *ActionManager) GetDeadletterQueueItems() ([]*model.Article, error) {
	return getQueueItems(self.path_dlq)
}

func (self *ActionManager) DeleteQueueItem(id *model.Id) error {
	f_path := filepath.Join(self.path_q, id.String())
	return os.Remove(f_path)
}

func (self *ActionManager) DeleteDeadletterQueueItem(id *model.Id) error {
	f_path := filepath.Join(self.path_dlq, id.String())
	return os.Remove(f_path)
}

func (self *ActionManager) Redrive(id *model.Id) error {
	q_fpath := filepath.Join(self.path_q, id.String())
	dlq_fpath := filepath.Join(self.path_dlq, id.String())
	return os.Rename(dlq_fpath, q_fpath)
}

func (self *ActionManager) CancelAction() {
	self.newSession()
}

func (self *ActionManager) Restart() {
	self.newSession()
}

func (self *ActionManager) newSession() task.Canceller {
	self.session_mtx.Lock()
	defer self.session_mtx.Unlock()

	if self.session_cc != nil {
		self.session_cc.Cancel()
	}
	self.session_cc = self.msn.NewCancel()
	return self.session_cc
}

func (self *ActionManager) getSessionCanceller() task.Canceller {
	self.session_mtx.Lock()
	defer self.session_mtx.Unlock()

	return self.session_cc
}

func (self *ActionManager) run_f_watcher() {
	go func(msn *task.Mission) {
		defer msn.Done()

		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			slog.Error("cannot make wathcer a dir %s: %s", self.path_q, err)
			return
		}
		defer watcher.Close()
		if err := watcher.Add(self.path_q); err != nil {
			slog.Error("cannot make wathcer a dir %s: %s", self.path_q, err)
			return
		}

		for {
			select {
			case <- msn.RecvCancel():
				return
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				slog.Error("cannot make wathcer a dir %s: %s", self.path_q, err)
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				go func(event *fsnotify.Event) {
					if event.Op&(fsnotify.Create) == 0 {
						return
					}
					slog.Debug("%p recv target: %s, %s", self, event.Name, event.Op)

					select {
					case <- msn.RecvCancel():
						return
					case self.fpath_ch <- filepath.Clean(event.Name):
					}
				}(&event)
			}
		}
	}(self.msn.New())

	go func (msn *task.Mission) {
		fs, err := os.ReadDir(self.path_wip)
		if err != nil {
			slog.Error("cannot read %s queue: %s", self.path_wip, err)
			return
		}

		for _, f := range fs {
			if f.IsDir() {
				continue
			}
			select {
			case <- msn.RecvCancel():
				return
			case self.fpath_ch <- filepath.Join(self.path_wip, f.Name()):
			}
		}
	}(self.msn.New())

	go func (msn *task.Mission) {
		fs, err := os.ReadDir(self.path_q)
		if err != nil {
			slog.Error("cannot read %s queue: %s", self.path_q, err)
			return
		}

		for _, f := range fs {
			if f.IsDir() {
				continue
			}
			select {
			case <- msn.RecvCancel():
				return
			case self.fpath_ch <- filepath.Join(self.path_q, f.Name()):
			}
		}
	}(self.msn.New())
}

func (self *ActionManager) task_handler(msn *task.Mission) {
	defer msn.Done()

	for {
		select {
		case <- msn.RecvCancel():
		case q_fpath := <- self.fpath_ch:
			func(msn *task.Mission) {
				defer msn.Done()

				cc := self.getSessionCanceller()
				go func () {
					select {
					case <- msn.RecvDone():
					case <- msn.RecvCancel():
					case <- cc.RecvCancel():
						msn.Cancel()
					}
				}()

				if _, err := os.Stat(q_fpath); os.IsNotExist(err) {
					self.logger.Warn("%s is not exist", q_fpath)
					return
				}

				fname := filepath.Base(q_fpath)
				wip_f_path := filepath.Join(self.path_wip, fname)
				if err := os.Rename(q_fpath, wip_f_path); err != nil {
					self.logger.Error("Cannot mv wip file: %s -> %s: %s", q_fpath, wip_f_path, err)
					return
				}

				if err := self.action.Do(msn.New(), self.logger, wip_f_path); err != nil {
					self.logger.Error("%s Action Failed: %s", self.action.Name(), err)

					dlq_f_path := filepath.Join(self.path_dlq, fname)
					slog.Debug("mv %s %s", wip_f_path, dlq_f_path)
					if err := os.Rename(wip_f_path, dlq_f_path); err != nil {
						slog.Error("cannot move to dlq: src: %s, dst: %s, err: %s", wip_f_path, dlq_f_path, err)
					}
					return
				}
				if err := os.Remove(wip_f_path); err != nil {
					slog.Error("cannot rm q file: %s, err: %s", wip_f_path, err)
				}
			}(msn.New())
		}
	}
}

func getQueueItems(path string) ([]*model.Article, error) {
	fs, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	articles := []*model.Article{}
	for _, f := range fs {
		if f.IsDir() {
			continue
		}

		f_path := filepath.Join(path, f.Name())
		content, err := os.ReadFile(f_path)
		if err != nil {
			return nil, fmt.Errorf("failed: cannot read q file: %s, %s", f_path, err)
		}
		var ex_artcle external.Article
		if err := json.Unmarshal(content, &ex_artcle); err != nil {
			return nil, fmt.Errorf("failed: cannot convert q file: %s, %s", f_path, err)
		}
		article, err := model.ImportExternalArticle(&ex_artcle)
		if err != nil {
			return nil, fmt.Errorf("failed: cannot convert q file: %s, %s", f_path, err)
		}

		articles = append(articles, article)
	}
	return articles, nil
}
