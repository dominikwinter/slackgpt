package openai

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/dominikwinter/slackgpt/internal/client/helper"
	"github.com/imroc/req/v3"
)

type I = map[string]interface{}
type S = map[string]string

type Message struct {
	Id      string    `json:"id"`
	Content []Content `json:"content"`
}

type Messages struct {
	Data []Message `json:"data"`
}

type Content struct {
	Text Text `json:"text"`
}

type Text struct {
	Value string `json:"value"`
}

type Run struct {
	Id     string `json:"id"`
	Status string `json:"status"`
}

type Thread struct {
	Id string `json:"id"`
}

type Client struct {
	Client *req.Client
}

func New(url, token, organization string) *Client {
	if url == "" {
		panic("url not set")
	}

	if token == "" {
		panic("token not set")
	}

	if organization == "" {
		panic("organization not set")
	}

	return &Client{
		Client: req.C().
			// EnableDumpAll().
			SetBaseURL(url).
			SetUserAgent("github.com/dominikwinter/slackgpt").
			SetCommonHeader("Accept", "application/json").
			SetCommonHeader("Content-Type", "application/json").
			SetCommonHeader("OpenAI-Organization", organization).
			SetCommonHeader("OpenAI-Beta", "assistants=v1").
			SetCommonBearerAuthToken(token).
			SetTimeout(20 * time.Second). // OpenAI API can be slow
			SetCookieJar(nil).
			SetCommonErrorResult(&helper.ErrorMessage{}),
	}
}

// Create thread
// https://platform.openai.com/docs/api-reference/threads/createThread
func (c *Client) CreateThread() (res *Thread, err error) {
	_, err = c.Client.R().
		SetSuccessResult(&res).
		Post("/threads")
	return
}

// Create message
// https://platform.openai.com/docs/api-reference/messages/createMessage
func (c *Client) CreateMessage(threadId, content string) (res *Message, err error) {
	_, err = c.Client.R().
		SetBody(I{"role": "user", "content": content}).
		SetSuccessResult(&res).
		SetPathParam("threadId", threadId).
		Post("/threads/{threadId}/messages")
	return
}

// List messages
// https://platform.openai.com/docs/api-reference/messages/listMessages
func (c *Client) ListMessages(threadId, messageIId string) (res *Messages, err error) {
	_, err = c.Client.R().
		SetSuccessResult(&res).
		SetQueryParams(S{"before": messageIId}).
		SetPathParam("threadId", threadId).
		Get("/threads/{threadId}/messages")
	return
}

// Create run
// https://platform.openai.com/docs/api-reference/runs/createRun
func (c *Client) CreateRun(threadId string) (res *Run, err error) {
	_, err = c.Client.R().
		SetBody(I{"assistant_id": os.Getenv("OPENAI_ASSISTANTS_ID")}).
		SetSuccessResult(&res).
		SetPathParam("threadId", threadId).
		Post("/threads/{threadId}/runs")
	return
}

// Retrieve run
// https://platform.openai.com/docs/api-reference/runs/getRun
// status: queued, in_progress, requires_action, cancelling, cancelled, failed, completed, expired
func (c *Client) WaitForRunCompleted(threadId, runId string) (res *Run, err error) {
	_, err = c.Client.R().
		SetSuccessResult(&res).
		SetPathParam("threadId", threadId).
		SetPathParam("runId", runId).
		SetRetryCount(20).
		// SetRetryFixedInterval(500 * time.Millisecond).
		SetRetryFixedInterval(2 * time.Second).
		SetRetryCondition(func(res *req.Response, err error) bool {
			if err != nil {
				return false
			}

			run, ok := res.SuccessResult().(*Run)
			if !ok {
				return false
			}

			switch run.Status {
			case "queued", "in_progress":
				return true
			default:
				return false
			}
		}).
		Get("/threads/{threadId}/runs/{runId}")

	if res.Status != "completed" {
		return nil, fmt.Errorf("failed to wait for run completed: %s", res.Status)
	}

	return
}

// process OpenAI's Q'n'A flow, which is totally over-engineered
func (c *Client) SendMessageAndWaitForAnswer(threadId, content string) (string, error) {
	message, err := c.CreateMessage(threadId, content)
	if err != nil {
		return "", fmt.Errorf("failed to create message: %w", err)
	}

	run, err := c.CreateRun(threadId)
	if err != nil {
		return "", fmt.Errorf("failed to create run: %w", err)
	}

	_, err = c.WaitForRunCompleted(threadId, run.Id)
	if err != nil {
		return "", fmt.Errorf("failed to get run: %w", err)
	}

	messages, err := c.ListMessages(threadId, message.Id)
	if err != nil {
		return "", fmt.Errorf("failed to list messages: %w", err)
	}

	if len(messages.Data) == 0 {
		return "", fmt.Errorf("failed to get response from OpenAI: no messages")
	}

	answers := make([]string, 0)

	for _, message := range messages.Data {
		answers = append(answers, message.Content[0].Text.Value)
	}

	// reverse
	sort.Slice(answers, func(i, j int) bool { return true })

	return strings.Join(answers, "\n\n"), nil
}
