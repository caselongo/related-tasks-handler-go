package tasks

import (
	"errors"
	"fmt"
	"golang.org/x/exp/maps"
	"strings"
	"sync"
)

type Task struct {
	Id      string
	WaitFor []string
	Skip    bool
}

type task struct {
	waitFor []string
	started bool
	done    bool
}

type Handler struct {
	sync.Mutex
	handlerFunc func(string) error
	tasks       map[string]task
	errors      []error
	wg          sync.WaitGroup
}

func (h *Handler) tryStart(id string) bool {
	defer h.Unlock()
	h.Lock()

	t, ok := h.tasks[id]
	if !ok {
		return false
	}

	if t.done {
		return false
	}

	if t.started {
		return false
	}

	if len(t.waitFor) > 0 {
		for _, w := range t.waitFor {
			tw, _ := h.tasks[w]
			if !tw.done {
				return false
			}
		}
	}

	t.started = true
	h.tasks[id] = t

	return true
}

func (h *Handler) setDone(id string) {
	defer h.Unlock()
	h.Lock()

	q := h.tasks[id]
	q.done = true
	h.tasks[id] = q
}

func (h *Handler) get(id string) (task, bool) {
	defer h.Unlock()
	h.Lock()

	q, ok := h.tasks[id]

	return q, ok
}

func (h *Handler) set(name string, t task) {
	defer h.Unlock()
	h.Lock()

	h.tasks[name] = t
}

func (h *Handler) all() map[string]task {
	defer h.Unlock()
	h.Lock()

	return h.tasks
}

func (h *Handler) execute(id string) error {
	_, ok := h.get(id)
	if !ok {
		return errors.New(fmt.Sprintf("task with id '%s' does not exist", id))
	}

	return h.handlerFunc(id)
}

func (h *Handler) Run() error {
	err := h.startAllTasks()
	if err != nil {
		return err
	}

	h.wg.Wait()

	return nil
}

func (h *Handler) startAllTasks() error {
	for id := range h.all() {
		if !h.tryStart(id) {
			continue
		}

		h.wg.Add(1)

		idScoped := id

		go func() {
			defer func() {
				h.setDone(idScoped)
				err := h.startAllTasks()
				if err != nil {
					h.errors = append(h.errors, err)
					return
				}
				h.wg.Done()
			}()

			err := h.execute(idScoped)
			if err != nil {
				h.errors = append(h.errors, err)
				return
			}
		}()
	}

	if len(h.errors) > 0 {
		return h.errors[0]
	}

	return nil
}

func NewHandler(handlerFunc func(string) error, tasks ...Task) (*Handler, error) {
	if handlerFunc == nil {
		return nil, errors.New("no task handlerFunc function provided")
	}

	var handler = Handler{
		handlerFunc: handlerFunc,
		tasks:       make(map[string]task),
	}

	var canStart = false
	var waitForItself []string

f:
	for _, t := range tasks {
		_, ok := handler.tasks[t.Id]
		if ok {
			return nil, errors.New(fmt.Sprintf("multiple tasks with id '%s'", t.Id))
		}

		for _, w := range t.WaitFor {
			if t.Id == w {
				waitForItself = append(waitForItself, t.Id)
				continue f
			}
		}

		handler.tasks[t.Id] = task{
			waitFor: t.WaitFor,
			started: t.Skip,
			done:    t.Skip,
		}

		if !canStart {
			if len(t.WaitFor) == 0 {
				canStart = true
			}
		}
	}

	if !canStart {
		return nil, errors.New("tasks must include at least one task that has not to wait for other tasks")
	}

	if len(waitForItself) > 0 {
		return nil, errors.New(fmt.Sprintf("tasks cannot wit for themselves, please check task(s) '%s'", strings.Join(waitForItself, "','")))
	}

	var notExisting = make(map[string]bool)

	for _, t := range handler.all() {
		for _, w := range t.waitFor {
			_, ok := handler.get(w)
			if !ok {
				notExisting[w] = true
			}
		}
	}

	if len(notExisting) > 0 {
		return nil, errors.New(fmt.Sprintf("the following tasks do not exist but are referred to in waitFor of other tasks : %s", strings.Join(maps.Keys(notExisting), ", ")))
	}

	return &handler, nil
}
