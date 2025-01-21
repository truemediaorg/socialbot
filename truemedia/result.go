package truemedia

/*
The shape changes somewhat depending on State:
If State == ERROR:

	No Scores, AnalysisTime, or Pending. Just Errors.

If STATE == PROCESSING:

	Scores contains scores from completed models
	AnalysisTime is the currently-elapsed processing time
	Pending contains a list of models yet to complete
	Errors is empty

If State == COMPLETE:

	Scores contains scortes from the models
	AnalysisTime is the elapsed time of processing
	Pending and Errors are empty
*/
type AnalysisState string

const (
	AnalysisStateProcessing AnalysisState = "PROCESSING"
	AnalysisStateComplete   AnalysisState = "COMPLETE"
	AnalysisStateError      AnalysisState = "ERROR"
)

type GetResultResponse struct {
	State        AnalysisState          `json:"state"`
	Scores       map[string]interface{} `json:"scores,omitempty"`
	Verdict      Verdict                `json:"verdict,omitempty"`
	AnalysisTime float32                `json:"analysisTime,omitempty"`
	Pending      []string               `json:"pending,omitempty"`
	Errors       []string               `json:"errors,omitempty"`
}
