package responder

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/g8rswimmer/go-twitter/v2"
	"github.com/lucsky/cuid"
	"github.com/truemediaorg/socialbot/database/db"
	"github.com/truemediaorg/socialbot/model"
	"github.com/truemediaorg/socialbot/truemedia"
	twitterutil "github.com/truemediaorg/socialbot/twitter"

	log "github.com/sirupsen/logrus"
)

const (
	// Used in the "final" response
	lowVerdictMsg       = "🟢 TrueMedia verdict: 𝗹𝗶𝘁𝘁𝗹𝗲 𝗲𝘃𝗶𝗱𝗲𝗻𝗰𝗲 of manipulation. More analysis >"
	uncertainVerdictMsg = "🟡 TrueMedia verdict: 𝘀𝗼𝗺𝗲 𝗲𝘃𝗶𝗱𝗲𝗻𝗰𝗲 of manipulation. More analysis >"
	highVerdictMsg      = "🔴 TrueMedia verdict: 𝘀𝘂𝗯𝘀𝘁𝗮𝗻𝘁𝗶𝗮𝗹 𝗲𝘃𝗶𝗱𝗲𝗻𝗰𝗲 of manipulation. More analysis >"
	unknownVerdictMsg   = "⏳ TrueMedia is taking longer than usual to analyze this media. Results will be available >"
	trueMediaTagline    = "TrueMedia detects political deepfakes in social media. It's non-profit, non-partisan, and free."
	userThankMsg        = "Thank you for submitting this, @%s." // UserName goes in the %s

	maximumProcessingDelay = 15 * time.Minute // How long to wait before posting the analysis URL anyway

	// Copied from the Twitter response, beware the risk of this changing over time.
	deletedPostErrorMsg   = "You attempted to reply to a Tweet that is deleted or not visible to you."
	duplicatePostErrorMsg = "You are not allowed to create a Tweet with duplicate content."
)

type ReplyHandler interface {
	GetMentionsNeedingRepliesForPlatform(ctx context.Context, platform model.Platform) ([]model.Mention, error)
	DeleteMention(ctx context.Context, mentionID string) error
	AddReply(ctx context.Context, mentionID string, platform model.Platform, platformID string, replyType db.ReplyType) error
	FindRepliesForMention(ctx context.Context, mentionID string) ([]model.Reply, error)
	GetMediaPostUrl(ctx context.Context, mediaID string) (string, error)
}

type TweetResponder interface {
	TweetResponse(ctx context.Context, replyToID string, message string) (*twitter.CreateTweetResponse, error)
}

type MediaAnalyzer interface {
	GetAnalysis(mediaID string) (*truemedia.GetResultResponse, error)
}

type Responder struct {
	twitterService   TweetResponder
	truemediaService MediaAnalyzer
	db               ReplyHandler
	testModeEnabled  bool
}

func NewResponder(twitterService TweetResponder, truemediaService MediaAnalyzer, db ReplyHandler, isTestMode bool) *Responder {
	return &Responder{
		twitterService:   twitterService,
		truemediaService: truemediaService,
		db:               db,
		testModeEnabled:  isTestMode,
	}
}

func (r *Responder) Respond(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			log.Debug("exiting Responder by closing channel")
			return nil
		case <-time.After(5 * time.Second): // check for work every 5 seconds to avoid slamming the truemedia API
			mentions, err := r.db.GetMentionsNeedingRepliesForPlatform(ctx, model.PlatformX)
			if err != nil {
				log.Errorf("error getting work: %v", err)
				return err
			}
			if len(mentions) > 0 {
				log.Infof("found %d mentions needing replies", len(mentions))
			}

			for _, mention := range mentions {
				if mention.MediaID == "" {
					// TODO: pop it from the list for next time, the media must've been deleted in the DB
					log.WithField("ID", mention.PlatformID).Warn("Mention missing media; was media deleted?")
					continue
				}
				analysis, err := r.truemediaService.GetAnalysis(mention.MediaID)
				if err != nil {
					log.Errorf("error getting analysis: %v", err)
					continue
				}
				switch analysis.State {
				case truemedia.AnalysisStateComplete:
					log.Infof("analysis complete for %s, responding to %s post ID=%s", mention.MediaID, mention.Platform, mention.PlatformID)
					err := r.respondToPostWithAnalysis(ctx, mention, *analysis)
					if err != nil {
						var apiError *twitter.ErrorResponse
						if errors.As(err, &apiError) {
							r.handleAPIError(ctx, mention, *apiError)
						} else {
							log.Errorf("error responding to post: %v", err)
						}
					}
				case truemedia.AnalysisStateProcessing:
					log.WithField("verdict", analysis.Verdict).Infof("%s still processing, continuing...", mention.MediaID)
					// If the Media stays in "Processing" for too long, respond with a link to the incomplete analysis
					if time.Since(mention.Enqueued) > maximumProcessingDelay {
						log.WithField("mediaId", mention.MediaID).WithField("enqueued", mention.Enqueued).Warnf("analysis taking too long, responding anyway")
						err := r.respondToPostWithAnalysis(ctx, mention, *analysis)
						if err != nil {
							var apiError *twitter.ErrorResponse
							if errors.As(err, &apiError) {
								r.handleAPIError(ctx, mention, *apiError)
							} else {
								log.Errorf("error responding to post: %v", err)
							}
						}
					}
				case truemedia.AnalysisStateError:
					log.Errorf("errors analyzing media %v: %v", mention.MediaID, analysis.Errors)
				}
			}
		}
	}
}

func (r *Responder) respondToPostWithAnalysis(ctx context.Context, mention model.Mention, analysis truemedia.GetResultResponse) error {
	responseContent := generateResponseContent(mention, analysis)
	if responseContent == "" {
		return fmt.Errorf("failed to generate response for media %s with rank %s", mention.MediaID, analysis.Verdict)
	}

	// Reply to the Media message
	parentPostURL, err := r.db.GetMediaPostUrl(ctx, mention.MediaID)
	if err != nil {
		return err
	}

	_, replyToID, err := twitterutil.DeconstructTweetURL(parentPostURL)
	if err != nil {
		return err
	}

	var replyID string
	if r.testModeEnabled {
		replyID = cuid.New()
		log.WithField("replyToID", replyToID).WithField("responseContent", responseContent).Infof("Simulating reply to %s with post ID %s", mention.Platform, replyID)
	} else {
		resp, err := r.twitterService.TweetResponse(ctx, replyToID, responseContent)
		if err != nil {
			return err
		}
		replyID = resp.Tweet.ID
	}

	err = r.db.AddReply(ctx, mention.ID, mention.Platform, replyID, db.ReplyTypeFinal)
	if err != nil {
		log.Warnf("Reply %s posted to %s but wasn't recorded in the database", replyID, model.PlatformX)
		return err
	}
	return nil
}

func (r *Responder) handleAPIError(ctx context.Context, mention model.Mention, apiError twitter.ErrorResponse) {
	if apiError.Detail == deletedPostErrorMsg {
		// The post with the media is deleted--there's nothing to
		// reply to, so delete the mention from the queue.
		err := r.db.DeleteMention(ctx, mention.ID)
		if err != nil {
			log.WithField("id", mention.ID).WithField("mediaId", mention.MediaID).Errorf("Caught error removing deleted post from database: %v", err.Error())
		} else {
			log.WithField("id", mention.ID).WithField("mediaId", mention.MediaID).Warn("Deleted post detected. Removing mention from database.")
		}
	} else if apiError.Detail == duplicatePostErrorMsg {
		// There's already a reply, but it isn't recorded in the database.
		// Add a record with a fake ID so the bot doesn't get hung up on this.
		err := r.db.AddReply(ctx, mention.ID, mention.Platform, cuid.New(), db.ReplyTypeFinal)
		if err != nil {
			log.WithField("id", mention.ID).WithField("mediaId", mention.MediaID).Errorf("Error inserting missing reply record: %v", err.Error())
		} else {
			log.WithField("id", mention.ID).WithField("mediaId", mention.MediaID).Warn("Missing Reply record detected. Adding a new record.")
		}
	} else {
		log.WithField("statusCode", apiError.StatusCode).WithField("title", apiError.Title).Errorf("API error responding to post: %v", apiError.Detail)
	}
}

func generateResponseContent(mention model.Mention, analysis truemedia.GetResultResponse) string {
	var verdictMsg string
	if analysis.State == truemedia.AnalysisStateComplete {
		verdictMsg = describeVerdict(analysis.Verdict)
	} else if analysis.State == truemedia.AnalysisStateProcessing {
		verdictMsg = unknownVerdictMsg
	}
	if verdictMsg == "" {
		// we didn't get a low/uncertain/high
		// TODO: something-went-wrong response
		return ""
	}
	userMention := fmt.Sprintf(userThankMsg, mention.PlatformUserName)
	return fmt.Sprintf("%s\n%s\n\n%s\n\n%s", verdictMsg, generateResultsURL(mention.MediaID), userMention, trueMediaTagline)
}

func generateResultsURL(mediaID string) string {
	return fmt.Sprintf("https://OPEN-TODO-PLACEHOLDER/media/analysis?id=%s", mediaID)
}

func describeVerdict(verdict truemedia.Verdict) string {
	switch verdict {
	case truemedia.VerdictTrusted:
		fallthrough
	case truemedia.VerdictLow:
		return lowVerdictMsg
	case truemedia.VerdictUncertain:
		return uncertainVerdictMsg
	case truemedia.VerdictHigh:
		return highVerdictMsg
	default:
		return ""
	}
}
