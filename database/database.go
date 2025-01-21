package database

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lucsky/cuid"
	"github.com/truemediaorg/socialbot/database/db"
	"github.com/truemediaorg/socialbot/model"
)

type Database struct {
	connString string
	pool       *pgxpool.Pool
}

func NewDatabase(connString string) *Database {
	return &Database{
		connString: connString,
	}
}

func (d *Database) Connect(ctx context.Context) error {
	var err error
	d.pool, err = pgxpool.New(ctx, d.connString)
	if err != nil {
		return err
	}
	return nil
}

func (d *Database) Disconnect() {
	d.pool.Close()
}

func (d *Database) AddMention(ctx context.Context, platformID string, platformUserName string, platform model.Platform, mediaID string) error {
	// don't really care about the result, as long as this succeeds
	_, err := d.pool.Exec(ctx, `
	INSERT INTO mention_queue (id, platform, platform_id, platform_user_name, enqueued, media_id) VALUES ($1, $2, $3, $4, $5, $6)`,
		cuid.New(),
		platform,
		platformID,
		platformUserName,
		time.Now().UTC(), // the DB stores timezones and assumes UTC
		mediaID,
	)
	if err != nil {
		return err
	}
	return nil
}

func (d *Database) DeleteMention(ctx context.Context, mentionID string) error {
	// don't really care about the result, as long as this succeeds
	_, err := d.pool.Exec(ctx, `
	DELETE FROM mention_queue WHERE id = $1`,
		mentionID,
	)
	if err != nil {
		return err
	}
	return nil
}

func (d *Database) GetLatestTweetID(ctx context.Context) (string, error) {
	// For Twitter, IDs are always increasing
	var id string
	// TODO: using model.PlatformX here is blurring the lines between model and db
	err := d.pool.QueryRow(
		ctx,
		`SELECT 
			platform_id 
		FROM mention_queue 
		WHERE platform = $1
		ORDER BY platform_id DESC
		LIMIT 1`,
		model.PlatformX,
	).Scan(&id)
	if err != nil {
		// A blank table is OK and obviously can't return rows
		if err == pgx.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return id, nil
}

func (d *Database) GetMentionsNeedingRepliesForPlatform(ctx context.Context, platform model.Platform) ([]model.Mention, error) {
	var mentions []model.Mention
	var raws []db.MentionQueue
	rows, err := d.pool.Query(ctx, `
	SELECT 
		id, 
		platform, 
		platform_id,
		platform_user_name,
		media_id, 
		enqueued 
	FROM mention_queue
	WHERE 
		id NOT IN ( 
			SELECT mention_id 
			FROM mention_reply 
			WHERE platform = $1
			  AND type = 'FINAL' 
		) 
		AND platform = $2
	ORDER BY enqueued DESC`,
		platform,
		platform,
	)
	if err != nil {
		return nil, err
	}

	raws, err = pgx.CollectRows(rows, pgx.RowToStructByName[db.MentionQueue])
	if err != nil {
		return nil, err
	}

	for _, raw := range raws {
		mention, err := model.MentionFromMentionQueue(raw)
		if err != nil {
			return nil, err
		}
		mentions = append(mentions, *mention)
	}

	return mentions, nil
}

func (d *Database) AddReply(ctx context.Context, mentionID string, platform model.Platform, platformID string, replyType db.ReplyType) error {
	_, err := d.pool.Exec(
		ctx,
		`INSERT INTO mention_reply (id, mention_id, platform, platform_id, replied, type) VALUES ($1, $2, $3, $4, $5, $6)`,
		cuid.New(),
		mentionID,
		platform,
		platformID,
		time.Now().UTC(), // the DB stores timezones and assumes UTC
		replyType,
	)
	if err != nil {
		return err
	}
	return nil
}

func (d *Database) FindRepliesForMention(ctx context.Context, mentionID string) ([]model.Reply, error) {
	var replies []model.Reply
	var raws []db.MentionReply
	rows, err := d.pool.Query(ctx, `
	SELECT 
		id, 
		mention_id,
		platform, 
		platform_id, 
		replied, 
		type
	FROM mention_reply
	WHERE mention_id = $1`,
		mentionID,
	)
	if err != nil {
		return nil, err
	}

	raws, err = pgx.CollectRows(rows, pgx.RowToStructByName[db.MentionReply])
	if err != nil {
		return nil, err
	}

	for _, raw := range raws {
		reply, err := model.ReplyFromMentionReply(raw)
		if err != nil {
			return nil, err
		}
		replies = append(replies, *reply)
	}

	return replies, nil
}

func (d *Database) GetMediaPostUrl(ctx context.Context, mediaID string) (string, error) {
	var postUrl string
	err := d.pool.QueryRow(ctx, `
		SELECT pm.post_url 
		FROM media m
		JOIN post_media pm ON m.id = pm.media_id
		WHERE m.id = $1
		LIMIT 1`,
		mediaID,
	).Scan(&postUrl)
	if err != nil {
		return "", err
	}
	return postUrl, nil
}
