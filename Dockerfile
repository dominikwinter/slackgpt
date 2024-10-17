FROM golang:1.23-alpine AS base

WORKDIR /usr/src/slackgpt

COPY . .

RUN go mod download
RUN go mod verify

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -trimpath -o ./slackgpt

FROM scratch

COPY --from=base /usr/src/slackgpt/slackgpt /usr/bin/slackgpt
COPY --from=base /etc/passwd /etc/passwd
COPY --from=base /etc/group /etc/group

USER nobody:nobody

CMD ["slackgpt"]
