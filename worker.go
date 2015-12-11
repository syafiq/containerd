package containerd

import (
	"sync"
	"time"

	"github.com/docker/containerd/runtime"
)

type Worker interface {
	Start()
}

type StartTask struct {
	Container  runtime.Container
	Checkpoint string
	IO         *runtime.IO
	Err        chan error
}

func NewWorker(s *Supervisor, wg *sync.WaitGroup) Worker {
	return &worker{
		s:  s,
		wg: wg,
	}
}

type worker struct {
	wg *sync.WaitGroup
	s  *Supervisor
}

func (w *worker) Start() {
	defer w.wg.Done()
	for t := range w.s.tasks {
		started := time.Now()
		// start logging the container's stdio
		l, err := w.s.log(t.Container.Path(), t.IO)
		if err != nil {
			evt := NewEvent(DeleteEventType)
			evt.ID = t.Container.ID()
			w.s.SendEvent(evt)
			t.Err <- err
			continue
		}
		w.s.containers[t.Container.ID()].logger = l
		if t.Checkpoint != "" {
			if err := t.Container.Restore(t.Checkpoint); err != nil {
				evt := NewEvent(DeleteEventType)
				evt.ID = t.Container.ID()
				w.s.SendEvent(evt)
				t.Err <- err
				continue
			}
		} else {
			if err := t.Container.Start(); err != nil {
				evt := NewEvent(DeleteEventType)
				evt.ID = t.Container.ID()
				w.s.SendEvent(evt)
				t.Err <- err
				continue
			}
		}
		ContainerStartTimer.UpdateSince(started)
		t.Err <- nil
	}
}