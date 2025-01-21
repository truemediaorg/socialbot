package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dghubble/oauth1"
	"github.com/truemediaorg/socialbot/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	twauth "github.com/dghubble/oauth1/twitter"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	oauthConfig oauth1.Config
)

func init() {
	rootCmd.AddCommand(authorizerCmd)
}

var authorizerCmd = &cobra.Command{
	Use:   "authorizer",
	Short: "Generates a key/secret pair for socialbot to post as a user",
	Long:  `Generates a key/secret pair for socialbot to post as a user`,
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.FromEnvfile()

		log.SetLevel(cfg.LogLevel)
		switch cfg.LogFormat {
		case config.LogFormatJSON:
			log.SetFormatter(&log.JSONFormatter{})
		default:
			log.SetFormatter(&log.TextFormatter{})
		}

		awsConfig, err := awsconfig.LoadDefaultConfig(context.Background())
		if err != nil {
			log.Panic(err)
		}
		secretsManagerClient := secretsmanager.NewFromConfig(awsConfig)

		// Get the Twitter secrets from AWS Secrets Manager
		result, err := secretsManagerClient.GetSecretValue(context.Background(), &secretsmanager.GetSecretValueInput{SecretId: aws.String(cfg.Twitter.SecretPath)})
		if err != nil {
			log.Panic(err.Error())
		}
		var twitterSecrets config.TwitterSecretData
		err = json.Unmarshal([]byte(*result.SecretString), &twitterSecrets)
		if err != nil {
			log.Panicf("twitter secrets read error: %v", err)
		}

		oauthConfig = oauth1.Config{
			ConsumerKey:    twitterSecrets.ConsumerKey,
			ConsumerSecret: twitterSecrets.ConsumerSecret,
			CallbackURL:    "oob",
			Endpoint:       twauth.AuthorizeEndpoint,
		}

		requestToken, err := login()
		if err != nil {
			log.Fatalf("Request Token Phase: %s", err.Error())
		}
		accessToken, err := receivePIN(requestToken)
		if err != nil {
			log.Fatalf("Access Token Phase: %s", err.Error())
		}

		fmt.Println("Consumer was granted an access token to act on behalf of a user.")
		fmt.Printf("token: %s\nsecret: %s\n", accessToken.Token, accessToken.TokenSecret)
	},
}

// These are lifted from the oauth1 library's twitter PIN example
// https://github.com/dghubble/oauth1/blob/main/examples/twitter-login.go

func login() (requestToken string, err error) {
	requestToken, _, err = oauthConfig.RequestToken()
	if err != nil {
		return "", err
	}
	authorizationURL, err := oauthConfig.AuthorizationURL(requestToken)
	if err != nil {
		return "", err
	}
	fmt.Printf("Open this URL in your browser:\n%s\n", authorizationURL.String())
	return requestToken, err
}

func receivePIN(requestToken string) (*oauth1.Token, error) {
	fmt.Printf("Paste your PIN here: ")
	var verifier string
	_, err := fmt.Scanf("%s", &verifier)
	if err != nil {
		return nil, err
	}
	// Twitter ignores the oauth_signature on the access token request. The user
	// to which the request (temporary) token corresponds is already known on the
	// server. The request for a request token earlier was validated signed by
	// the consumer. Consumer applications can avoid keeping request token state
	// between authorization granting and callback handling.
	accessToken, accessSecret, err := oauthConfig.AccessToken(requestToken, "secret does not matter", verifier)
	if err != nil {
		return nil, err
	}
	return oauth1.NewToken(accessToken, accessSecret), err
}
