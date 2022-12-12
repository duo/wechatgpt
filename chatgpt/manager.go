package chatgpt

import (
	"context"
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
	client *ChatGPT

	taskQueue     map[string](chan *Task)
	taskQueueLock sync.Mutex
}

func NewTaskManager(email, password, sessionToken, userAgent, cfClearance string) *TaskManager {
	return &TaskManager{
		client:    NewChatGPT(email, password, sessionToken, userAgent, cfClearance),
		taskQueue: make(map[string](chan *Task)),
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

			conversation := tm.client.NewConversation("")

			for task := range queue {
				log.Debugf("Handle Task: %+v", task)

				// Handle command
				if task.content == cmdReset {
					conversation = tm.client.NewConversation("")
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
