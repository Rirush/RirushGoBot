package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/inconshreveable/go-keen"
)

const (
	envtoken = "TG_TOKEN"
	envurl   = "URL"
)

var (
	keenClient = &keen.Client{WriteKey: os.Getenv("KEEN_MASTER"), ProjectID: os.Getenv("KEEN_PROJECT")}
	query      = make(chan tgbotapi.MessageConfig)
)

type messageEvent struct {
	UserID  int
	Message string
	Type    string
}

func initquery(bot *tgbotapi.BotAPI) {
	sender := time.NewTicker(time.Millisecond * 34)
	go func() {
		for range sender.C {
			bot.Send(<-query)
		}
	}()
}

func help() string {
	return "I can do this stuff:\n" +
		"Show this help message with /help\n" +
		"\nIf you found bug, report about it to [GitHub Issues](https://github.com/Rirush/RirushGoBot/issues)\n" +
		"If you have any ideas, you can tell me about them by creating [issue](https://github.com/Rirush/RirushGoBot/issues)"
}

func handle(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	if update.Message == nil {
		log.Printf("Update(%d) skipped because it isn't a message", update.UpdateID)
	}

	event := messageEvent{UserID: update.Message.From.ID, Message: update.Message.Text}

	log.Printf("Message(%d) with text %q received from @%s:%d", update.UpdateID, update.Message.Text, update.Message.From.UserName, update.Message.From.ID)

	if update.Message.IsCommand() {
		log.Printf("Message(%d) is command", update.UpdateID)
		event.Type = "command"
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
		msg.ParseMode = "markdown"
		msg.DisableWebPagePreview = true

		switch strings.ToLower(update.Message.Command()) {

		case "start":
			msg.Text = fmt.Sprintf("Hello, %s.\n"+
				"I'm a bot built with [telegram-bot-api](https://github.com/go-telegram-bot-api/telegram-bot-api) and [Golang](https://golang.org).\n"+
				"If you're interested in stuff that I can do - use /help.\n\n"+
				"Also, I'm open-source, so you can find my source code [here](https://github.com/Rirush/RirushGoBot)", update.Message.From.FirstName)

		case "help":
			msg.Text = help()
		}

		query <- msg
	} else {
		event.Type = "message"
	}
	keenClient.AddEvent("messages", &event)
}

// Running bot with webhook updates, status page, database and Keen.io analytics
func production() {
	bot, _ := tgbotapi.NewBotAPI(os.Getenv(envtoken))
	bot.Debug = false

	initquery(bot)

	bot.SetWebhook(tgbotapi.NewWebhook(os.Getenv(envurl) + os.Getenv(envtoken)))

	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	// Status "page"
	router.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "UP")
	})
	router.POST("/", func(c *gin.Context) {
		c.String(http.StatusOK, "UP")
	})

	updates := make(chan tgbotapi.Update, bot.Buffer)

	router.POST("/"+os.Getenv(envtoken), func(c *gin.Context) {
		bytes, _ := ioutil.ReadAll(c.Request.Body)

		var update tgbotapi.Update
		json.Unmarshal(bytes, &update)

		updates <- update

		c.Status(http.StatusOK)
	})

	go router.Run("0.0.0.0:" + os.Getenv("PORT"))

	for update := range updates {
		go handle(bot, update)
	}
}

// Running only bot w/o any Heroku-related things
func development() {
	bot, err := tgbotapi.NewBotAPI(os.Getenv(envtoken))

	// "Handle" error
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true

	log.Printf("Using account @%s", bot.Self.UserName)

	initquery(bot)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 100

	updates, err := bot.GetUpdatesChan(u)

	// Another error "handler"
	if err != nil {
		log.Panic(err)
	}

	log.Printf("Starting messages polling")

	// Messages polling and handling
	for update := range updates {
		go handle(bot, update)
	}
}

func main() {
	if os.Getenv(envtoken) == "" {
		log.Panic("TG_TOKEN isn't set.")
	}
	if os.Getenv("PRODUCTION") == "1" {
		production()
	} else {
		development()
	}
}
