package truemedia

type ResolveMediaRequest struct {
	PostURL string `json:"postUrl"`
}

type ResolveMediaStatus string

const (
	ResolveMediaStatusResolved ResolveMediaStatus = "resolved"
	ResolveMediaStatusFailed   ResolveMediaStatus = "failed"
)

type ResolveMediaItem struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	MimeType string `json:"mimeType"`
}

/*
This shape changes somewhat depending on Result:
If Result == failed:

	FailureReason and FailureDetails are populated, Media is empty

If Result == resolved:

	Media is populated, FailureReason and FailureDetails empty.
*/
type ResolveMediaResponse struct {
	Result         string             `json:"result"`
	Media          []ResolveMediaItem `json:"media,omitempty"`
	FailureReason  string             `json:"reason,omitempty"`
	FailureDetails string             `json:"details,omitempty"`
}
