package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"

	"github.com/dghubble/oauth1"
	"github.com/truemediaorg/socialbot/config"
	"golang.org/x/exp/maps"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/g8rswimmer/go-twitter/v2"
	log "github.com/sirupsen/logrus"
)

type TwitterService struct {
	userID      string
	apiClient   *twitter.Client
	oauthClient *twitter.Client

	timelinePageSize int
}

type authorize struct {
	Token string
}

func (a authorize) Add(req *http.Request) {
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", a.Token))
}

func NewTwitterService(ctx context.Context, cfg config.Config, secretsManagerClient *secretsmanager.Client) *TwitterService {
	// Get the Twitter secrets from AWS Secrets Manager
	result, err := secretsManagerClient.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{SecretId: aws.String(cfg.Twitter.SecretPath)})
	if err != nil {
		log.Fatal(err.Error())
	}
	var twitterSecrets config.TwitterSecretData
	err = json.Unmarshal([]byte(*result.SecretString), &twitterSecrets)
	if err != nil {
		log.Panicf("twitter secrets read error: %v", err)
	}

	// Initialize the API Client (used for most API calls)
	apiClient := &twitter.Client{
		Authorizer: authorize{
			Token: twitterSecrets.BearerToken,
		},
		Client: http.DefaultClient,
		Host:   "https://api.twitter.com",
	}

	// Resolve the user ID of the bot
	users, err := apiClient.UserNameLookup(ctx, []string{cfg.Twitter.BotUserName}, twitter.UserLookupOpts{})
	if err != nil {
		log.Panicf("user lookup error: %v", err)
	}
	log.Debugf("user lookup rate limit---limit=%d;remaining=%d;reset=%d", users.RateLimit.Limit, users.RateLimit.Remaining, users.RateLimit.Reset)

	if len(users.Raw.Users) == 0 {
		log.Panicf("user not found: %s", cfg.Twitter.BotUserName)
	}

	// Initialize the OAuth Client (used for making OAuth-authenticated API calls)
	oauthConfig := oauth1.NewConfig(twitterSecrets.ConsumerKey, twitterSecrets.ConsumerSecret)
	oauthToken := oauth1.NewToken(twitterSecrets.AccessToken, twitterSecrets.AccessTokenSecret)
	oauthClient := &twitter.Client{
		Authorizer: &authorize{},
		Client:     oauthConfig.Client(ctx, oauthToken),
		Host:       "https://api.twitter.com",
	}
	return &TwitterService{
		userID:           users.Raw.Users[0].ID,
		apiClient:        apiClient,
		oauthClient:      oauthClient,
		timelinePageSize: cfg.Twitter.TimelinePageSize,
	}
}

/*
Gets all mentions from the Twitter API since a given tweet ID.
If sinceID is empty, returns all available mentions.
This has the capability to return a lot of mentions over multiple requests and may take some time to return.
Returned tweets are re-sorted to oldest-first so processing always happens starting with the oldest posts.
*/
func (s *TwitterService) GetAllTimelineMentionsSince(ctx context.Context, sinceID string) ([]*twitter.TweetDictionary, error) {
	paginationToken := ""
	tweets := map[string]*twitter.TweetDictionary{}
	for ok := true; ok; ok = (paginationToken != "") {
		apiOpts := twitter.UserMentionTimelineOpts{
			TweetFields:     []twitter.TweetField{twitter.TweetFieldAuthorID, twitter.TweetFieldConversationID, twitter.TweetFieldAttachments},
			MediaFields:     []twitter.MediaField{twitter.MediaFieldMediaKey, twitter.MediaFieldType, twitter.MediaFieldURL},
			UserFields:      []twitter.UserField{twitter.UserFieldUserName},
			Expansions:      []twitter.Expansion{twitter.ExpansionReferencedTweetsID, twitter.ExpansionAttachmentsMediaKeys, twitter.ExpansionAuthorID, twitter.ExpansionInReplyToUserID},
			MaxResults:      s.timelinePageSize,
			PaginationToken: paginationToken,
			SinceID:         sinceID,
		}

		log.WithField("sinceID", sinceID).WithField("paginationToken", paginationToken).WithField("userID", s.userID).Info("requesting timeline mentions")
		timeline, err := s.apiClient.UserMentionTimeline(ctx, s.userID, apiOpts)
		if err != nil {
			return nil, err
		}
		paginationToken = timeline.Meta.NextToken
		log.WithField("paginationToken", paginationToken).Debug("new pagination token")
		log.WithField("limit", timeline.RateLimit.Limit).WithField("remaining", timeline.RateLimit.Remaining).WithField("reset", timeline.RateLimit.Reset).Info("rate limit data for timeline mentions")
		for key, value := range timeline.Raw.TweetDictionaries() {
			// Shouldn't have to worry about collisions since these IDs are unique
			if tweets[key] != nil {
				log.Warnf("Duplicate tweet found while getting mentions: %v", key)
			}
			tweets[key] = value
		}
	}
	// Sort the tweets from oldest to newest (ascending ID)
	tweetSlice := maps.Values(tweets)
	sort.Slice(tweetSlice, func(i, j int) bool {
		return tweetSlice[i].Tweet.ID < tweetSlice[j].Tweet.ID
	})
	return tweetSlice, nil
}

func (s *TwitterService) TweetResponse(ctx context.Context, replyToID string, message string) (*twitter.CreateTweetResponse, error) {
	return s.oauthClient.CreateTweet(ctx, twitter.CreateTweetRequest{
		Text: message,
		Reply: &twitter.CreateTweetReply{
			InReplyToTweetID: replyToID,
		},
	})
}

func (s *TwitterService) UserID() string {
	return s.userID
}
