FROM golang:1.22 as build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY ./ ./

RUN CGO_ENABLED=0 GOOS=linux go build .

FROM alpine:3 as run

COPY --from=build src/socialbot /socialbot

# There's a single HTTP endpoint for healthchecks
EXPOSE 8080

# Note: .env file expected; mount it at runtime
ENTRYPOINT ["/socialbot", "server"]

FROM run as prod

RUN <<EOF cat >> .env
POSTGRES_SECRETS_PATH=socialbot/prod/postgres

TWITTER_SECRETS_PATH=socialbot/prod/twitter
TWITTER_USERNAME=OPEN-TODO-PLACEHOLDER
TWITTER_TIMELINE_PAGE_SIZE=25

TRUEMEDIA_SECRETS_PATH=socialbot/prod/truemedia
TRUEMEDIA_API=OPEN-TODO-PLACEHOLDER/api
TRUEMEDIA_RESULTS_INTERVAL=5
TRUEMEDIA_RESOLVE_INTERVAL=60

LOG_LEVEL=info
LOG_FORMAT=json
EOF

ENTRYPOINT ["/socialbot", "server"]
