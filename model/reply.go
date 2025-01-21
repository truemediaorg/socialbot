package model

import (
	"time"

	"github.com/truemediaorg/socialbot/database/db"
)

type Reply struct {
	ID         string
	MentionID  string
	Platform   Platform
	PlatformID string
	Replied    time.Time
	MediaID    string
	Type       db.ReplyType
}

func ReplyFromMentionReply(mq db.MentionReply) (*Reply, error) {
	platform, err := ParsePlatform(mq.Platform)
	if err != nil {
		return nil, err
	}
	return &Reply{
		ID:         mq.ID,
		MentionID:  mq.MentionID,
		Platform:   platform,
		PlatformID: mq.PlatformID,
		Replied:    mq.Replied,
		Type:       mq.Type,
	}, nil
}
