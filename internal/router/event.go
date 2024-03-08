package router

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/dominikwinter/slackgpt/internal/client/openai"
	"github.com/dominikwinter/slackgpt/internal/client/slack"
	"github.com/gofiber/fiber/v3"
)

type SlackRequestBody struct {
	Type      string `json:"type"`
	Challenge string `json:"challenge"`
	Event     *Event `json:"event"`
}

type Event struct {
	Type        string       `json:"type"`
	Ts          string       `json:"ts"`
	Channel     string       `json:"channel"`
	ThreadTs    string       `json:"thread_ts"`
	EventTs     string       `json:"event_ts"`
	User        string       `json:"user"`
	BotId       string       `json:"bot_id"`
	Text        string       `json:"text"`
	UserProfile *UserProfile `json:"user_profile"`
}

type UserProfile struct {
	RealName string `json:"real_name"`
}

var slackAiThreadMap = map[string]string{}
var DEFAULT_ERROR_MESSAGE = ":exploding_head: Sorry, sometimes i'm forgetful. Please start another thread."

var slackClient = slack.New(os.Getenv("SLACK_API_URL"), os.Getenv("SLACK_BOT_TOKEN"))
var openaiClient = openai.New(os.Getenv("OPENAI_API_URL"), os.Getenv("OPENAI_API_KEY"), os.Getenv("OPENAI_ORGANIZATION"))

func initChat(event *Event) error {
	_, err := slackClient.AddReactions(event.Channel, "thinking", event.Ts)
	if err != nil {
		return fmt.Errorf("failed to add reaction: %w", err)
	}
	defer slackClient.DelReactions(event.Channel, "thinking", event.Ts)

	openAiThread, err := openaiClient.CreateThread()
	if err != nil {
		slackClient.AddToThread(DEFAULT_ERROR_MESSAGE, event.Channel, event.Ts)
		return fmt.Errorf("failed to create thread: %w", err)
	}

	slackAiThreadMap[event.Ts] = openAiThread.Id

	message := fmt.Sprintf(`
		Parse the "Text" and extract the name of the colleague as feedback receiver. The name is in the format like <@U069DBU1TGQ>.
		If you can't find a name, please ask the user to provide the name of the colleague.
		After that start with the questionnaire.

		Text: %s
	`, event.Text)

	openAiAnswer, err := openaiClient.SendMessageAndWaitForAnswer(openAiThread.Id, message)
	if err != nil {
		slackClient.AddToThread(DEFAULT_ERROR_MESSAGE, event.Channel, event.Ts)
		return fmt.Errorf("failed to send message and wait for answer: %w", err)
	}

	if _, err := slackClient.StartThread(openAiAnswer, event.Channel, event.Ts, openAiThread.Id); err != nil {
		slackClient.AddToThread(DEFAULT_ERROR_MESSAGE, event.Channel, event.Ts)
		return fmt.Errorf("failed to start thread: %w", err)
	}

	return nil
}

func replyChat(event *Event) error {
	openAiThreadId := slackAiThreadMap[event.ThreadTs]

	// in case of restart `slackAiThreadMap` is of course empty, so we need to
	// get the aiThreadId from the slack history.
	if openAiThreadId == "" {
		// get second message from thread
		history, err := slackClient.GetHistory(event.Channel, event.ThreadTs, 1)
		if err != nil {
			slackClient.AddToThread(DEFAULT_ERROR_MESSAGE, event.Channel, event.Ts)
			return fmt.Errorf("failed to get history: %w", err)
		}

		openAiThreadId = getOpenAiThreadIdFromSecondMessageFromThread(history)
		if openAiThreadId == "" {
			slackClient.AddToThread(DEFAULT_ERROR_MESSAGE, event.Channel, event.ThreadTs)
			return fmt.Errorf("failed to get openAiThreadId from second message from thread")
		}

		openAiThreadId = "thread_" + openAiThreadId
		slackAiThreadMap[event.ThreadTs] = openAiThreadId
	}

	slackClient.AddReactions(event.Channel, "thinking", event.Ts)
	defer slackClient.DelReactions(event.Channel, "thinking", event.Ts)

	openAiAnswer, err := openaiClient.SendMessageAndWaitForAnswer(openAiThreadId, event.Text)
	if err != nil {
		slackClient.AddToThread(DEFAULT_ERROR_MESSAGE, event.Channel, event.Ts)
		return fmt.Errorf("failed to send message and wait for answer: %w", err)
	}

	if _, err := slackClient.AddToThread(openAiAnswer, event.Channel, event.Ts); err != nil {
		slackClient.AddToThread(DEFAULT_ERROR_MESSAGE, event.Channel, event.Ts)
		return fmt.Errorf("failed to start thread: %w", err)
	}

	return nil
}

// JS: history.messages?.[1]?.blocks?.at(-1)?.elements?.[0]?.text
func getOpenAiThreadIdFromSecondMessageFromThread(history *slack.History) string {
	if len(history.Messages) > 1 {
		message := history.Messages[1]

		if len(message.Blocks) > 0 {
			block := message.Blocks[len(message.Blocks)-1]

			if len(block.Elements) > 0 {
				element := block.Elements[0]

				return element.Text
			}
		}
	}

	return ""
}

func Setup(app *fiber.App, log *slog.Logger) {
	app.Post("/api/v1/events", func(c fiber.Ctx) error {
		c.Response().Header.SetContentType("plain/text; charset=utf-8")

		var body SlackRequestBody

		if err := c.Bind().JSON(&body); err != nil {
			return err
		}

		log.Info("Incoming Request", "body", body)

		// url body request, used only once on app configuration in slack
		// https://api.slackClient.com/apps/xxxxxx/event-subscriptions
		// https://api.slackClient.com/apis/connections/events-api#handshake
		if body.Type == "url_verification" {
			log.Info("Responding to url verification", "challenge", body.Challenge)
			return c.SendString(body.Challenge)
		}

		if body.Type != "event_callback" ||
			body.Event == nil ||
			body.Event.Type != "message" ||
			body.Event.BotId != "" ||
			body.Event.UserProfile == nil {
			return c.SendString("ok")
		}

		go func() {
			var err error

			if body.Event.ThreadTs == "" {
				// direct message, not in thread, init chat
				err = initChat(body.Event)
			} else {
				// if message is from user and we are in the thread .. so it's obviously the second message.
				err = replyChat(body.Event)
			}

			if err != nil {
				log.Error("Failed to process event", "error", err)
			}
		}()

		return c.SendString("ok")
	})
}
