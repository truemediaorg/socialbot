package watcher

import (
	"context"
	"time"

	"github.com/g8rswimmer/go-twitter/v2"
	"github.com/truemediaorg/socialbot/database"
	"github.com/truemediaorg/socialbot/model"
	"github.com/truemediaorg/socialbot/service"
	twitterutil "github.com/truemediaorg/socialbot/twitter"

	log "github.com/sirupsen/logrus"
)

type Watcher struct {
	twitterService   *service.TwitterService
	truemediaService *service.TruemediaService
	db               *database.Database
}

func NewWatcher(twitterService *service.TwitterService, truemediaService *service.TruemediaService, db *database.Database) *Watcher {
	return &Watcher{
		twitterService:   twitterService,
		truemediaService: truemediaService,
		db:               db,
	}
}

func (w *Watcher) Watch(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			log.Debug("exiting Watcher by closing channel")
			return nil
		case <-time.After(5 * time.Minute): // check for mentions every 5 minutes because of low rate limit
			latestTweetID, err := w.db.GetLatestTweetID(ctx)
			if err != nil {
				// TODO: better handling if DB connection falters?
				return err
			}
			tweets, err := w.twitterService.GetAllTimelineMentionsSince(ctx, latestTweetID)
			if err != nil {
				if rateLimit, ok := twitter.RateLimitFromError(err); ok {
					// If we hit the rate limit, sleep until it resets and try again
					log.WithField("limit", rateLimit.Limit).WithField("remaining", rateLimit.Remaining).Warnf("X rate limit encountered, sleeping for %fs", time.Until(rateLimit.Reset.Time()).Seconds())
					time.Sleep(time.Until(rateLimit.Reset.Time()))
					continue
				}
				return err
			}
			for _, tweet := range tweets {
				tweetID := tweet.Tweet.ID
				tweetAuthor := tweet.Author.UserName
				var mediaTweetReference *twitter.TweetReference
				log.Debugf("tweet %s has %d referenced tweets", tweetID, len(tweet.ReferencedTweets))
				for _, referencedTweet := range tweet.ReferencedTweets {
					log.WithField("referenceType", referencedTweet.Reference.Type).Debug()
					if twitterutil.IsReplyReference(referencedTweet) && twitterutil.TweetHasMedia(referencedTweet.TweetDictionary.Tweet) {
						mediaTweetReference = referencedTweet
						break
					}
				}
				if mediaTweetReference == nil {
					// If there's no media, just move on to the next mention
					continue
				}
				log.WithField("tweet", mediaTweetReference.TweetDictionary.Tweet).Debug("tweet ID")
				log.WithField("tweetAuthor", tweetAuthor).Debug("tweet author")
				log.WithField("tweetID", mediaTweetReference.TweetDictionary.Tweet.ID).Info("found tweet with media")
				mediaTweetURL := twitterutil.ConstructTweetURL(mediaTweetReference.TweetDictionary.Author.UserName, mediaTweetReference.TweetDictionary.Tweet.ID)
				log.WithField("mediaTweetURL", mediaTweetURL).Infof("resolving X post ID=%s", mediaTweetReference.TweetDictionary.Tweet.ID)
				mediaID, err := w.truemediaService.ResolvePostMedia(mediaTweetURL)
				if err != nil {
					log.Errorf("error resolving post media: %v", err)
					continue // HACK: skip this one and move on for now
				}
				// Ask for results immediately so analysis begins
				if results, err := w.truemediaService.GetAnalysis(mediaID); err != nil {
					log.WithField("mediaID", mediaID).Errorf("error starting analysis: %v", err)
					// This doesn't stop the presses for this piece of media because the Responder also calls this,
					// it'll just take longer for the bot to respond with results.
				} else {
					log.WithField("mediaID", mediaID).Debugf("initial results: %v", results)
				}
				if err := w.db.AddMention(ctx, tweetID, tweetAuthor, model.PlatformX, mediaID); err != nil {
					log.Errorf("error adding post to database: %v", err)
					// Context canceled errors are expected if the program is terminating, so stop the loop in that case
					if ctx.Err() == context.Canceled {
						return err
					}
				}
				// hacky way to avoid hitting the resolve rate limit
				time.Sleep(w.truemediaService.ResolveInterval())
			}
		}
	}
}
