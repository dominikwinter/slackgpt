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

// https://platform.openai.com/docs/api-reference/files/object
type File struct {
	Id        string `json:"id"`
	Object    string `json:"object"`
	Bytes     int    `json:"bytes"`
	CreatedAt int    `json:"created_at"`
	Filename  string `json:"filename"`
	Purpose   string `json:"purpose"`
}

// https://platform.openai.com/docs/api-reference/assistants/object
type Assistant struct {
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
		SetHeader("Accept", "application/json").
		SetHeader("Content-Type", "application/json").
		SetSuccessResult(&res).
		Post("/v1/threads")
	return
}

// Create message
// https://platform.openai.com/docs/api-reference/messages/createMessage
func (c *Client) CreateMessage(threadId, content string) (res *Message, err error) {
	_, err = c.Client.R().
		SetHeader("Accept", "application/json").
		SetHeader("Content-Type", "application/json").
		SetBody(S{"role": "user", "content": content}).
		SetSuccessResult(&res).
		SetPathParam("threadId", threadId).
		Post("/v1/threads/{threadId}/messages")
	return
}

// List messages
// https://platform.openai.com/docs/api-reference/messages/listMessages
func (c *Client) ListMessages(threadId, messageIId string) (res *Messages, err error) {
	_, err = c.Client.R().
		SetHeader("Accept", "application/json").
		SetSuccessResult(&res).
		SetQueryParams(S{"before": messageIId}).
		SetPathParam("threadId", threadId).
		Get("/v1/threads/{threadId}/messages")
	return
}

// Create run
// https://platform.openai.com/docs/api-reference/runs/createRun
func (c *Client) CreateRun(threadId string) (res *Run, err error) {
	_, err = c.Client.R().
		SetHeader("Accept", "application/json").
		SetHeader("Content-Type", "application/json").
		SetBody(S{"assistant_id": os.Getenv("OPENAI_ASSISTANTS_ID")}).
		SetSuccessResult(&res).
		SetPathParam("threadId", threadId).
		Post("/v1/threads/{threadId}/runs")
	return
}

// Retrieve run
// https://platform.openai.com/docs/api-reference/runs/getRun
// status: queued, in_progress, requires_action, cancelling, cancelled, failed, completed, expired
func (c *Client) WaitForRunCompleted(threadId, runId string) (res *Run, err error) {
	_, err = c.Client.R().
		SetHeader("Accept", "application/json").
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
		Get("/v1/threads/{threadId}/runs/{runId}")

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

// Upload file
// https://platform.openai.com/docs/api-reference/files/create
func (c *Client) UploadFile(paramName, filePath string) (res *File, err error) {
	_, err = c.Client.R().
		SetHeader("Accept", "application/json").
		SetFile(paramName, filePath).
		SetSuccessResult(&res).
		SetFormData(S{"purpose": "assistants"}).
		Post("/v1/files")
	return
}

// Create assistant
// https://platform.openai.com/docs/api-reference/assistants/createAssistant
func (c *Client) CreateAssistant(name, instructions, model, toolType string, fileIds []string) (res *Assistant, err error) {
	_, err = c.Client.R().
		SetHeader("Accept", "application/json").
		SetHeader("Content-Type", "application/json").
		SetBody(I{
			"name":         name,
			"instructions": instructions,
			"model":        model,
			"tools":        []interface{}{S{"type": toolType}},
			"file_ids":     fileIds,
		}).
		SetSuccessResult(&res).
		Post("/v1/assistants")
	return
}
