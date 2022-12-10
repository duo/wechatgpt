package main

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/duo/wechatgpt/chatgpt"

	"github.com/eatmoreapple/openwechat"
	"github.com/skip2/go-qrcode"

	log "github.com/sirupsen/logrus"
)

const (
	qrCodeUrlPrefix    = "https://login.weixin.qq.com/l/"
	defaultTaskTimeout = 120 * time.Second
)

var (
	autoAccept  bool
	taskTimeout time.Duration
)

func main() {
	autoAccept = strings.ToLower(os.Getenv("AUTO_ACCEPT")) == "true"

	timeout := os.Getenv("TASK_TIMEOUT")
	if timeout == "" {
		taskTimeout = defaultTaskTimeout
	} else {
		duration, err := time.ParseDuration(timeout)
		if err != nil {
			log.Fatal(err)
		}
		taskTimeout = duration
	}

	sessionToken := os.Getenv("SESSION_TOKEN")
	if sessionToken == "" {
		log.Fatal("SESSION_TOKEN is empty")
	}

	taskManager := chatgpt.NewTaskManager(sessionToken)

	bot := openwechat.DefaultBot(openwechat.Desktop)

	bot.MessageHandler = func(msg *openwechat.Message) {
		handleMesasge(msg, taskManager)
	}

	bot.UUIDCallback = func(uuid string) {
		if runtime.GOOS == "windows" {
			openwechat.PrintlnQrcodeUrl(uuid)
		} else {
			q, _ := qrcode.New(qrCodeUrlPrefix+uuid, qrcode.Low)
			fmt.Println(q.ToString(true))
		}
	}

	reloadStorage := openwechat.NewJsonFileHotReloadStorage("storage.json")

	err := bot.HotLogin(reloadStorage)
	if err != nil {
		if err = bot.Login(); err != nil {
			log.Fatalf("login error: %v", err)
		}
	}

	bot.Block()
}

func handleMesasge(msg *openwechat.Message, taskManager *chatgpt.TaskManager) {
	if msg.IsFriendAdd() {
		if autoAccept {
			_, err := msg.Agree("I'm a ChatGPT bot~")
			if err != nil {
				log.Warnf("Faild to agree friend request: %v", err)
			}
		}
		return
	}

	if msg.IsSendBySelf() || (msg.IsSendByGroup() && !msg.IsAt()) || !msg.IsText() {
		return
	}

	log.Debugf("Receive msg: %s", msg.Content)

	content := strings.TrimSpace(msg.Content)
	responsePrefix := ""

	sender, err := msg.Sender()
	if err != nil {
		log.Warnf("Failed to get message sender: %v", err)
		if _, err := msg.ReplyText("[ERROR] Failed to get message sender"); err != nil {
			log.Warnf("Failed to reply: %v", err)
		}
		return
	}

	if msg.IsSendByGroup() {
		groupSender, err := msg.SenderInGroup()
		if err != nil {
			log.Warnf("Failed to get group sender: %v", err)
			if _, err := msg.ReplyText("[ERROR] Failed to get group sender"); err != nil {
				log.Warnf("Failed to reply: %v", err)
			}
			return
		}
		responsePrefix = "@" + groupSender.NickName + " "

		target := "@" + sender.Self.NickName
		content = strings.TrimSpace(strings.ReplaceAll(msg.Content, target, ""))
	}

	// Skip empty content
	if content == "" {
		return
	}

	taskManager.SendTask(chatgpt.NewTask(
		sender.ID(),
		content,
		taskTimeout,
		func(resp string, err error) {
			if err != nil {
				log.Warnf("Failed to get ChatGPT response: %v", err)
				if _, err := msg.ReplyText("[ERROR] Failed to get ChatGPT response"); err != nil {
					log.Warnf("Failed to reply: %v", err)
				}
			} else {
				log.Debugf("ChatGPT response: %s", resp)
				if _, err := msg.ReplyText(responsePrefix + resp); err != nil {
					log.Warnf("Failed to reply: %v", err)
				}
			}
		},
	))
}
