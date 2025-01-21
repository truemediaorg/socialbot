package twitter

type MediaType string

const (
	MediaTypePhoto       MediaType = "photo"
	MediaTypeAnimatedGIF MediaType = "animated_gif"
	MediaTypeVideo       MediaType = "video"
	// TODO: Round out the list
)

type TweetReferenceType string

const (
	TweetReferenceRepliedTo TweetReferenceType = "replied_to"
)
