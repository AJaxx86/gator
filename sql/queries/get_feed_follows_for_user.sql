-- name: GetFeedFollowsForUser :many
SELECT feeds.name, feeds.url
FROM feed_follows
JOIN feeds ON feed_follows.feed_id = feeds.id
WHERE feed_follows.user_id = $1;