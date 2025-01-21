package model

import (
	"time"

	"github.com/truemediaorg/socialbot/database/db"
)

type Mention struct {
	ID               string
	Platform         Platform
	PlatformID       string
	PlatformUserName string
	Enqueued         time.Time
	MediaID          string
}

func MentionFromMentionQueue(mq db.MentionQueue) (*Mention, error) {
	platform, err := ParsePlatform(mq.Platform)
	if err != nil {
		return nil, err
	}
	return &Mention{
		ID:               mq.ID,
		Platform:         platform,
		PlatformID:       mq.PlatformID,
		PlatformUserName: mq.PlatformUserName,
		Enqueued:         mq.Enqueued,
		MediaID:          mq.MediaID,
	}, nil
}
