package db

import "time"

type ReplyType string

const (
	ReplyTypeFinal      ReplyType = "FINAL"
	ReplyTypeProcessing ReplyType = "PROCESSING"
)

type MentionReply struct {
	ID         string    `db:"id"`
	MentionID  string    `db:"mention_id"`
	Platform   string    `db:"platform"`
	PlatformID string    `db:"platform_id"`
	Type       ReplyType `db:"type"`
	Replied    time.Time `db:"replied"`
}
