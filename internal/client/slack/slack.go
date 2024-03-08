package slack

import (
	"strconv"
	"strings"
	"time"

	"github.com/dominikwinter/slackgpt/internal/client/helper"
	"github.com/imroc/req/v3"
)

type I = map[string]interface{}
type S = map[string]string

type History struct {
	Messages []Message `json:"messages"`
}

type Message struct {
	Blocks []Block `json:"blocks"`
}

type Block struct {
	Elements []Element `json:"elements"`
}

type Element struct {
	Text string `json:"text"`
}

type Client struct {
	Client *req.Client
}

func New(url, token string) *Client {
	if url == "" {
		panic("url not set")
	}

	if token == "" {
		panic("token not set")
	}

	return &Client{
		Client: req.C().
			// EnableDumpAll().
			SetBaseURL(url).
			SetUserAgent("github.com/dominikwinter/slackgpt").
			SetCommonHeader("Accept", "application/json").
			SetCommonHeader("Content-Type", "application/json; charset=utf-8").
			SetCommonBearerAuthToken(token).
			SetTimeout(5 * time.Second).
			SetCookieJar(nil).
			SetCommonErrorResult(&helper.ErrorMessage{}),
	}
}

// https://api.slack.com/methods/chat.postMessage
func (c *Client) StartThread(text, channel, threadTs, aiThreadId string) (res *I, err error) {
	parts := strings.Split(strings.TrimSpace(text), "\n")
	var blocks []I

	for _, part := range parts {
		text := strings.TrimSpace(part)
		if text == "" {
			continue
		}

		blocks = append(blocks, I{"type": "section", "text": I{"type": "mrkdwn", "text": text}})
	}

	blocks = append(blocks, I{"type": "context", "elements": []I{{"type": "plain_text", "text": strings.Replace(aiThreadId, "thread_", "", 1)}}})

	_, err = c.Client.R().
		SetBody(I{
			"channel":   channel,
			"thread_ts": threadTs,
			"text":      text,
			"blocks":    blocks,
		}).
		SetSuccessResult(&res).
		Post("/chat.postMessage")
	return
}

// https://api.slack.com/methods/chat.postMessage
func (c *Client) AddToThread(text, channel, threadTs string) (res *I, err error) {
	_, err = c.Client.R().
		SetBody(I{
			"channel":   channel,
			"thread_ts": threadTs,
			"text":      text,
		}).
		SetSuccessResult(&res).
		Post("/chat.postMessage")
	return
}

// https://api.slack.com/methods/conversations.replies
func (c *Client) GetHistory(channel, threadTs string, limit int) (res *History, err error) {
	_, err = c.Client.R().
		SetQueryParams(S{
			"channel": channel,
			"ts":      threadTs,
			"limit":   strconv.Itoa(limit),
			"oldest":  threadTs,
		}).
		SetSuccessResult(&res).
		Get("/conversations.replies")
	return
}

// https://api.slack.com/methods/reactions.add
func (c *Client) AddReactions(channel, name, timestamp string) (res *I, err error) {
	_, err = c.Client.R().
		SetBody(I{
			"channel":   channel,
			"name":      name,
			"timestamp": timestamp,
		}).
		SetSuccessResult(&res).
		Post("/reactions.add")
	return
}

// https://api.slack.com/methods/reactions.remove
func (c *Client) DelReactions(channel, name, timestamp string) (res *I, err error) {
	_, err = c.Client.R().
		SetBody(I{
			"channel":   channel,
			"name":      name,
			"timestamp": timestamp,
		}).
		SetSuccessResult(&res).
		Post("/reactions.remove")
	return
}
