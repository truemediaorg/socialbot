package responder

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/g8rswimmer/go-twitter/v2"
	"github.com/truemediaorg/socialbot/database/db"
	"github.com/truemediaorg/socialbot/model"
	"github.com/truemediaorg/socialbot/truemedia"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockReplyHandler struct {
	mock.Mock
}

func (m *MockReplyHandler) GetMentionsNeedingRepliesForPlatform(ctx context.Context, platform model.Platform) ([]model.Mention, error) {
	args := m.Called(ctx, platform)
	return args.Get(0).([]model.Mention), args.Error(1)
}

func (m *MockReplyHandler) DeleteMention(ctx context.Context, mentionID string) error {
	args := m.Called(ctx, mentionID)
	return args.Error(0)
}

func (m *MockReplyHandler) AddReply(ctx context.Context, mentionID string, platform model.Platform, platformID string, replyType db.ReplyType) error {
	args := m.Called(ctx, mentionID, platform, platformID)
	return args.Error(0)
}

func (m *MockReplyHandler) FindRepliesForMention(ctx context.Context, mentionID string) ([]model.Reply, error) {
	args := m.Called(ctx, mentionID)
	return args.Get(0).([]model.Reply), args.Error(1)
}

func (m *MockReplyHandler) GetMediaPostUrl(ctx context.Context, mediaID string) (string, error) {
	args := m.Called(ctx, mediaID)
	return args.Get(0).(string), args.Error(1)
}

type MockTweetResponder struct {
	mock.Mock
}

func (m *MockTweetResponder) TweetResponse(ctx context.Context, replyToID string, message string) (*twitter.CreateTweetResponse, error) {
	args := m.Called(ctx, replyToID, message)
	return args.Get(0).(*twitter.CreateTweetResponse), args.Error(1)
}

type MockMediaAnalyzer struct {
	mock.Mock
}

func (m *MockMediaAnalyzer) GetAnalysis(mediaID string) (*truemedia.GetResultResponse, error) {
	args := m.Called(mediaID)
	return args.Get(0).(*truemedia.GetResultResponse), args.Error(1)
}

func TestDescribeVerdict(t *testing.T) {
	testCases := []struct {
		description string
		verdict     truemedia.Verdict
		startsWith  string
	}{
		{"high returns message with red circle", truemedia.VerdictHigh, "🔴"},
		{"uncertain returns message with yellow circle", truemedia.VerdictUncertain, "🟡"},
		{"low returns message with green circle", truemedia.VerdictLow, "🟢"},
		{"trusted is treated like low", truemedia.VerdictTrusted, "🟢"},
		{"unknown returns no message", truemedia.VerdictUnknown, ""},
	}
	for _, testCase := range testCases {
		t.Run(testCase.description, func(t *testing.T) {
			generatedMsg := describeVerdict(testCase.verdict)
			if len(testCase.startsWith) == 0 {
				assert.Zero(t, len(generatedMsg), "expected no content but got some")
			} else {
				assert.Greater(t, len(generatedMsg), 0, "expected message content but got none")
				msgFirstChar := string([]rune(generatedMsg)[0])
				assert.Equalf(t, testCase.startsWith, msgFirstChar, "expected message to begin with %s but found %s", testCase.startsWith, msgFirstChar)
			}
		})
	}
}

func TestRespondToPostWithAnalysis(t *testing.T) {
	t.Run("succeeds if all goes according to plan", func(t *testing.T) {
		mention := model.Mention{
			ID:               "c1123lfgdsa023",
			Platform:         model.PlatformX,
			PlatformID:       "123456",
			PlatformUserName: "foo",
			Enqueued:         time.Now(),
			MediaID:          "foo.mp4",
		}
		parentID := "789012"
		analysis := truemedia.GetResultResponse{
			State:   truemedia.AnalysisStateComplete,
			Verdict: truemedia.VerdictHigh,
		}
		parentReplyCreateResponse := twitter.CreateTweetResponse{Tweet: &twitter.CreateTweetData{ID: "66662222"}}

		mockTwitterService := new(MockTweetResponder)
		mockTwitterService.On("TweetResponse", context.TODO(), "789012", generateResponseContent(mention, analysis)).Return(&parentReplyCreateResponse, nil)
		mockDB := new(MockReplyHandler)
		mockDB.On("FindRepliesForMention", context.TODO(), mention.ID).Return([]model.Reply{}, nil)
		mockDB.On("GetMediaPostUrl", context.TODO(), mention.MediaID).Return(fmt.Sprintf("https://twitter.com/Foo/status/%s", parentID), nil)
		mockDB.On("AddReply", context.TODO(), mention.ID, mention.Platform, parentReplyCreateResponse.Tweet.ID).Return(nil)
		responder := Responder{
			twitterService:   mockTwitterService,
			truemediaService: new(MockMediaAnalyzer),
			db:               mockDB,
			testModeEnabled:  false,
		}

		err := responder.respondToPostWithAnalysis(context.TODO(), mention, analysis)
		assert.NoErrorf(t, err, "expected no error but got %v", err)
		mockTwitterService.AssertNumberOfCalls(t, "TweetResponse", 1)
		mockDB.AssertNumberOfCalls(t, "AddReply", 1)
	})

	t.Run("does not actually post if test mode is engaged", func(t *testing.T) {
		mention := model.Mention{
			ID:               "c1123lfgdsa023",
			Platform:         model.PlatformX,
			PlatformID:       "123456",
			PlatformUserName: "foo",
			Enqueued:         time.Now(),
			MediaID:          "foo.mp4",
		}
		parentID := "789012"
		analysis := truemedia.GetResultResponse{
			State:   truemedia.AnalysisStateComplete,
			Verdict: truemedia.VerdictHigh,
		}
		parentReplyCreateResponse := twitter.CreateTweetResponse{Tweet: &twitter.CreateTweetData{ID: "66662222"}}

		mockTwitterService := new(MockTweetResponder)
		mockTwitterService.On("TweetResponse", context.TODO(), "789012", generateResponseContent(mention, analysis)).Return(&parentReplyCreateResponse, nil)
		mockDB := new(MockReplyHandler)
		mockDB.On("FindRepliesForMention", context.TODO(), mention.ID).Return([]model.Reply{}, nil)
		mockDB.On("GetMediaPostUrl", context.TODO(), mention.MediaID).Return(fmt.Sprintf("https://twitter.com/Foo/status/%s", parentID), nil)
		mockDB.On("AddReply", context.TODO(), mention.ID, mention.Platform, mock.Anything).Return(nil)
		responder := Responder{
			twitterService:   mockTwitterService,
			truemediaService: new(MockMediaAnalyzer),
			db:               mockDB,
			testModeEnabled:  true,
		}

		err := responder.respondToPostWithAnalysis(context.TODO(), mention, analysis)
		assert.NoErrorf(t, err, "expected no error but got %v", err)
		mockTwitterService.AssertNumberOfCalls(t, "TweetResponse", 0)
		mockDB.AssertNumberOfCalls(t, "AddReply", 1)
	})

	t.Run("does not call AddReply if posting is unsuccessful", func(t *testing.T) {
		mention := model.Mention{
			ID:               "c1123lfgdsa023",
			Platform:         model.PlatformX,
			PlatformID:       "123456",
			PlatformUserName: "foo",
			Enqueued:         time.Now(),
			MediaID:          "foo.mp4",
		}
		parentID := "789012"
		analysis := truemedia.GetResultResponse{
			State:   truemedia.AnalysisStateComplete,
			Verdict: truemedia.VerdictHigh,
		}
		mockTwitterService := new(MockTweetResponder)
		mockTwitterService.On("TweetResponse", context.TODO(), "789012", generateResponseContent(mention, analysis)).Return(&twitter.CreateTweetResponse{}, fmt.Errorf("oh nooooo"))
		mockDB := new(MockReplyHandler)
		mockDB.On("FindRepliesForMention", context.TODO(), mention.ID).Return([]model.Reply{}, nil)
		mockDB.On("GetMediaPostUrl", context.TODO(), mention.MediaID).Return(fmt.Sprintf("https://twitter.com/Foo/status/%s", parentID), nil)
		responder := Responder{
			twitterService:   mockTwitterService,
			truemediaService: new(MockMediaAnalyzer),
			db:               mockDB,
			testModeEnabled:  false,
		}

		err := responder.respondToPostWithAnalysis(context.TODO(), mention, analysis)
		assert.Error(t, err, "expected error but got none")
		mockTwitterService.AssertNumberOfCalls(t, "TweetResponse", 1)
		mockDB.AssertNumberOfCalls(t, "AddReply", 0)
	})

	t.Run("responds with an unknown verdict message if the verdict is unknown when results must be posted", func(t *testing.T) {
		mention := model.Mention{
			ID:               "c1123lfgdsa023",
			Platform:         model.PlatformX,
			PlatformID:       "123456",
			PlatformUserName: "foo",
			Enqueued:         time.Now(),
			MediaID:          "foo.mp4",
		}
		parentID := "789012"
		// mock analysis response just needs a verdict
		analysis := truemedia.GetResultResponse{
			State:   truemedia.AnalysisStateProcessing,
			Verdict: truemedia.VerdictUnknown,
		}
		parentReplyCreateResponse := twitter.CreateTweetResponse{Tweet: &twitter.CreateTweetData{ID: "66662222"}}

		mockTwitterService := new(MockTweetResponder)
		mockTwitterService.On("TweetResponse", context.TODO(), "789012", generateResponseContent(mention, analysis)).Return(&parentReplyCreateResponse, nil)
		mockDB := new(MockReplyHandler)
		mockDB.On("FindRepliesForMention", context.TODO(), mention.ID).Return([]model.Reply{}, nil)
		mockDB.On("GetMediaPostUrl", context.TODO(), mention.MediaID).Return(fmt.Sprintf("https://twitter.com/Foo/status/%s", parentID), nil)
		mockDB.On("AddReply", context.TODO(), mention.ID, mention.Platform, parentReplyCreateResponse.Tweet.ID).Return(nil)
		responder := Responder{
			twitterService:   mockTwitterService,
			truemediaService: new(MockMediaAnalyzer),
			db:               mockDB,
			testModeEnabled:  false,
		}

		err := responder.respondToPostWithAnalysis(context.TODO(), mention, analysis)
		assert.NoErrorf(t, err, "expected no error but got %v", err)
		mockTwitterService.AssertNumberOfCalls(t, "TweetResponse", 1)
		mockDB.AssertNumberOfCalls(t, "AddReply", 1)
	})
}
