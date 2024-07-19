# Go Slack-Bot with OpenAI Assistants API

- [Slack Web API](https://api.slack.com/web)
- [Open AI Assistants API](https://platform.openai.com/docs/assistants/overview)


This is a Slack Bot written in Go. It integrates OpenAI’s latest Assistants API into Slack to respond to messages. The app acts as a web server that listens for incoming messages from Slack, sends these messages to OpenAI’s API, and subsequently sends the AI-generated responses back to Slack.

It uses the new Assistants API from OpenAI with the ability to upload files like PDFs, allowing you to ask questions about the content of the file.

Purposefully, OpenAI and Slack libs were foregone to create a more lightweight application.

Please note that you need to have a Slack account and a workspace to use this app. You also need to have an OpenAI account and an API key. Please edit the `.env` file with your Slack and OpenAI credentials.


## How to Start the Service

To start the SlackGPT service, follow these steps:

### Prerequisites

Ensure that you have the following installed on your local machine:
- [Go](https://golang.org/doc/install)
- [Docker](https://docs.docker.com/get-docker/)
- [Docker Compose](https://docs.docker.com/compose/install/)

### Building the Service

1. **Clone the Repository:**
   ```sh
   git clone https://github.com/yourusername/slackgpt.git
   cd slackgpt
   ```

2. **Build the Project:**
   Use the `Makefile` to build the project.
   ```sh
   make
   ```

3. **Release Build:**
   Create a release build for the service.
   ```sh
   make release
   ```

### Deploying the Service

1. **Copy Files to Server:**
   Manually copy the following files to your server using `scp` or any other file transfer method:
   - `slackgpt-linux-amd64`
   - `docker-compose.yml`
   - `.env`

   Example using `scp`:
   ```sh
   scp slackgpt-linux-amd64 docker-compose.yml .env youruser@yourserver:/path/to/destination/
   ```

2. **SSH into Your Server:**
   ```sh
   ssh youruser@yourserver
   ```

3. **Start the Service:**
   Navigate to the directory where you copied the files and run the following command to start the service using Docker Compose:
   ```sh
   docker compose up -d
   ```
