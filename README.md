# Go Slack-Bot with OpenAI Assistants API

- [Slack Web API](https://api.slack.com/web)
- [Open AI Assistants API](https://platform.openai.com/docs/assistants/overview)


This is a Slack Bot written in Go. It integrates OpenAI’s latest Assistants API into Slack to respond to messages. The app acts as a web server that listens for incoming messages from Slack, sends these messages to OpenAI’s API, and subsequently sends the AI-generated responses back to Slack.

It uses the new Assistants API from OpenAI with the ability to upload files like PDFs, allowing you to ask questions about the content of the file.

Purposefully, OpenAI and Slack libs were foregone to create a more lightweight application.

Please note that you need to have a Slack account and a workspace to use this app. You also need to have an OpenAI account and an API key. Please edit the `.env` file with your Slack and OpenAI credentials.
