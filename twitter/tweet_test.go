package twitter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeconstructTweetURL(t *testing.T) {
	t.Run("successfully parses twitter.com URLs", func(t *testing.T) {
		userName, tweetID, err := DeconstructTweetURL("https://www.twitter.com/FooBar/status/1234567")
		assert.NoError(t, err)
		assert.Equal(t, "FooBar", userName)
		assert.Equal(t, "1234567", tweetID)

		userName, tweetID, err = DeconstructTweetURL("http://www.twitter.com/FooBar/status/1234567")
		assert.NoError(t, err)
		assert.Equal(t, "FooBar", userName)
		assert.Equal(t, "1234567", tweetID)

		userName, tweetID, err = DeconstructTweetURL("https://twitter.com/FooBar/status/1234567")
		assert.NoError(t, err)
		assert.Equal(t, "FooBar", userName)
		assert.Equal(t, "1234567", tweetID)
	})

	t.Run("successfully parses x.com URLs", func(t *testing.T) {
		userName, tweetID, err := DeconstructTweetURL("https://www.x.com/FooBar/status/1234567")
		assert.NoError(t, err)
		assert.Equal(t, "FooBar", userName)
		assert.Equal(t, "1234567", tweetID)

		userName, tweetID, err = DeconstructTweetURL("http://www.x.com/FooBar/status/1234567")
		assert.NoError(t, err)
		assert.Equal(t, "FooBar", userName)
		assert.Equal(t, "1234567", tweetID)

		userName, tweetID, err = DeconstructTweetURL("https://x.com/FooBar/status/1234567")
		assert.NoError(t, err)
		assert.Equal(t, "FooBar", userName)
		assert.Equal(t, "1234567", tweetID)
	})

	t.Run("rejects non-Twitter URLs", func(t *testing.T) {
		userName, tweetID, err := DeconstructTweetURL("https://www.someotherwebsite.com/123456/status/foo")
		assert.Error(t, err)
		assert.Equal(t, "", userName)
		assert.Equal(t, "", tweetID)
	})
}
