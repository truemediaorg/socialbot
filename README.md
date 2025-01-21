> ⚠️ **WARNING:**
> This code is published as-is for reference and educational purposes in the field of deepfake detection. It represents a historical implementation by TrueMedia.org and is not actively maintained. The repository does not accept pull requests, issues, modifications, or support requests. The original TrueMedia.org organization has ceased operations.

# 🤖 TrueMedia.org socialbot

TrueMedia.org's interactive bot for Twitter/X, which is extensible to other social platforms. It watches for mentions in responses to Tweets with media, submits that media for deepfake analysis, and replies to that mention with the results.

## Initial setup

When setting up this project for the first time, you'll need to search the code for `OPEN-TODO-PLACEHOLDER` and replace with your specific service information.

## Getting Started

Make sure Golang is installed on your machine: https://go.dev/dl/

If you use VSCode, you'll want [the extension](https://marketplace.visualstudio.com/items?itemName=golang.Go). It'll prompt you to install some stuff later as well.

You can use the VSCode launch config to run main.go within VSCode.

This repo uses [Task](https://taskfile.dev/) for build tasks.

[Staticcheck](https://staticcheck.io) is used for linting and can be installed [using your favorite package manager](https://staticcheck.io/docs/getting-started/#distribution-packages)

### New to Golang?

A great starting place is the guided tour of the language: https://go.dev/tour/welcome/1

Go By Example is another good resource for answering "How do I...?" https://gobyexample.com/

Standard library documentation can be found here: https://pkg.go.dev/std

## Building / Development

`task format` will autoformat your code using `gofmt`, and `task lint` runs analysis tools. **CI will enforce linting rules.**

Build the binary using `task build`. `task build && ./socialbot server` will build and run the binary in one command.

Build and run the docker image using `task docker-run`, or use `task docker` to just build the image.

New images are automatically built and pushed to ECR when changes merged to `main` pass CI.

**Deployment is manual.** Go to the ECS service and force a redeploy.

## Database

Socialbot uses the same database as the TrueMedia.org API. Any necessary schema changes need to happen in the [truemediaorg/deepfake-app](https://github.com/truemediaorg/deepfake-app) repository first and be deployed **before** updated bot images can be deployed.

Documentation for the Postgres library `pgx` is here: https://pkg.go.dev/github.com/jackc/pgx/v5

## Configuration

Socialbot uses an envfile (`.env`) kept next to the binary to store configuration settings. Here's an example envfile with sample values:

```
POSTGRES_URL=postgres://mylocaluser:deepfake@localhost/mydatabase

# Path in AWS Secrets Manager where the Twitter credentials are found
TWITTER_SECRETS_PATH=socialbot/dev/twitter
# Username of the bot, for monitoring mentions
TWITTER_USERNAME=PLACEHOLDER_test

# Path in AWS Secrets Manager where the TrueMedia credentials are found
TRUEMEDIA_SECRETS_PATH=socialbot/dev/truemedia
# Host for the TrueMedia API
TRUEMEDIA_API=http://localhost:3000/api
# How long to wait between checking for results
TRUEMEDIA_RESULTS_INTERVAL=5
# How long to wait between submitting posts for resolution
TRUEMEDIA_RESOLVE_INTERVAL=60

# Minimum log level (set to "debug" for more verbosity)
# Will default to "info" if not present
LOG_LEVEL=info
# How to format the log messages
# Will default to "text" if not present
LOG_FORMAT=json
```

The envfile is used by both commands, so you will need it even if you just want to run `authorizer`.

**NOTE:** When running in Docker against a local truemedia API, use `host.docker.internal` instead of `localhost` everywhere in .env

### Secrets and Credentials

Production credentials are stored in AWS Secrets Manager under `socialbot/prod/` keys, and developer creds under the `socialbot/dev/` keys. These are used when running locally by ensuring `TWITTER_SECRETS_PATH` and `TRUEMEDIA_SECRETS_PATH` are pointing to `dev` paths.

`socialbot/prod/twitter` contains the five values needed for Twitter. `bearerToken,` `consumerKey`, and `consumerSecret` come from the Twitter application config. `accessToken` and `accessTokenSecret` are OAuth secrets used for posting tweets and are associated with the `PLACEHOLDER_test` account we use for posting in local testing environment.

`socialbot/prod/truemedia` contains the API key for the TrueMedia.org API.

`socialbot/prod/postgres` contains the secrets for postgres.

To set up an X account to post for testing:

- Update `TWITTER_USERNAME` in the .env file
- Use `authorizer` to generate the Access Token and Access Token Secret (see below), and update these two values in `socialbot/prod/twitter` inside AWS Secrets Manager.

### Testing

You'll use two Twitter accounts for testing. One is the aforementioned `PLACEHOLDER_test` account that plays the role of the bot. And you'll need a separate Twitter account to play the role of the user interacting with the bot. This second account does not need an API subscription or anything like that.

Using your test user account, post some media. Then reply to that post including a tag for the configured `TWITTER_USERNAME`. During the next iteration of the Watcher loop (every 5 minutes), the bot should find and process the post.

Note that the URL in the posted analysis link is hardcoded to the web app URL (see OPEN-TODO-PLACEHOLDER). When running the bot against a local TrueMedia service (as configured in the example above), these links won't work, and they won't have correct thumbnails.

## Usage

Ensure your AWS environment variables are set (e.g. `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY`), as the bot uses AWS Secrets Manager to get credentials for the X/Twitter and TrueMedia APIs.

There are also launch configurations provided for running either command in the VSCode debugger.

### Server

`socialbot server` runs the bot itself as a continuously-running process. It will start monitoring Twitter for mentions of the configured username.

### Authorizer

`socialbot authorizer` is a utility that generates the access token/secret pair for the account the bot should post under.

It will generate a URL for you to open in your browser. Ensuring you're signed in using the account you want the bot to use, follow the link to receive a PIN from Twitter.

Next, enter the PIN at the terminal prompt. The authorizer will complete the process with Twitter and output the secrets required to post.

```
% ./socialbot authorizer
Open this URL in your browser:
https://api.twitter.com/oauth/authorize?oauth_token=PLACEHOLDER
Paste your PIN here: PLACEHOLDER
Consumer was granted an access token to act on behalf of a user.
token: <token here>
secret: <secret here>
```

## Licenses

This project is licensed under the terms of the MIT license.

## Original Contributors

This TrueMedia.org service was built by [Michael Langan](https://github.com/mjlangan), with contributions from [Dawn Wright](https://github.com/DawnWright) and [Michael Bayne](https://github.com/samskivert).
