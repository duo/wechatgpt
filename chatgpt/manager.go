package chatgpt

import (
	"context"
	"net/http"
	"runtime/debug"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

const (
	queueCapacity = 1024

	cmdReset = "!reset"
)

type Task struct {
	id      string
	content string
	timeout time.Duration
	handler TaskHandler
}

type TaskHandler func(string, error)

func NewTask(id string, content string, timeout time.Duration, handler TaskHandler) *Task {
	return &Task{
		id:      id,
		content: content,
		timeout: timeout,
		handler: handler,
	}
}

type TaskManager struct {
	sessionToken string

	taskQueue     map[string](chan *Task)
	taskQueueLock sync.Mutex
}

func NewTaskManager(sessionToken string) *TaskManager {
	return &TaskManager{
		sessionToken: sessionToken,
		taskQueue:    make(map[string](chan *Task)),
	}
}

func (tm *TaskManager) SendTask(task *Task) {
	tm.taskQueueLock.Lock()
	defer tm.taskQueueLock.Unlock()

	queue, ok := tm.taskQueue[task.id]
	if !ok {
		queue = make(chan *Task, queueCapacity)
		tm.taskQueue[task.id] = queue

		go func() {
			defer func() {
				panicErr := recover()
				if panicErr != nil {
					log.Warnf("Panic while process %+v: %v\n%s", task, panicErr, debug.Stack())
				}
			}()

			httpClient := &http.Client{
				Transport: &http.Transport{
					Proxy: http.ProxyFromEnvironment,
				},
			}
			client := NewChatGPTWithClient(tm.sessionToken, httpClient)
			conversation := client.NewConversation("")

			for task := range queue {
				log.Debugf("Handle Task: %+v", task)

				// Handle command
				if task.content == cmdReset {
					conversation = client.NewConversation("")
					task.handler("Reset conversation done.", nil)
					continue
				}

				ctx, cancel := context.WithTimeout(context.Background(), task.timeout)

				resp, err := conversation.SendMessage(ctx, task.content)
				task.handler(resp, err)

				cancel()
			}
		}()
	}

	queue <- task
}
