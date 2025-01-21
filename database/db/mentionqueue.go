package db

import "time"

type MentionQueue struct {
	ID               string    `db:"id"`
	Platform         string    `db:"platform"`
	PlatformID       string    `db:"platform_id"`
	PlatformUserName string    `db:"platform_user_name"`
	MediaID          string    `db:"media_id"`
	Enqueued         time.Time `db:"enqueued"`
}
