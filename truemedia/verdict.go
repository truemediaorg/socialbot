package truemedia

type Verdict string

const (
	VerdictLow        Verdict = "low"
	VerdictTrusted    Verdict = "trusted"
	VerdictUncertain  Verdict = "uncertain"
	VerdictHigh       Verdict = "high"
	VerdictUnknown    Verdict = "unknown"
)
