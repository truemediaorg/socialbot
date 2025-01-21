package twitter

import (
	"errors"
	"fmt"
	"regexp"

	gotwitter "github.com/g8rswimmer/go-twitter/v2"
)

func ConstructTweetURL(authorName string, tweetID string) string {
	return fmt.Sprintf("https://twitter.com/%s/status/%s", authorName, tweetID)
}

// Takes in a URL and extracts the UserName and PostID if it's a Twitter/X URL.
// Return value order is UserName followed by PostID, followed by error.
func DeconstructTweetURL(tweetURL string) (string, string, error) {
	// regexp to capture username and ID out of a twitter post URL
	r := regexp.MustCompile(`^https?://(?:www\.)?(?:twitter|x).com/(?P<UserName>\w+)/status/(?P<UserID>\d+)`)
	isMatch := r.MatchString(tweetURL)
	if isMatch {
		matches := r.FindStringSubmatch(tweetURL)
		return matches[1], matches[2], nil
	}
	return "", "", errors.New("not a tweet URL")
}

func IsReplyReference(tweetRef *gotwitter.TweetReference) bool {
	return tweetRef.Reference.Type == string(TweetReferenceRepliedTo)
}

func TweetHasMedia(tweet gotwitter.TweetObj) bool {
	return tweet.Attachments != nil && (len(tweet.Attachments.MediaKeys) > 0)
}
