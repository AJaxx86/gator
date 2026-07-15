-- name: ListFeeds :many
SELECT feeds.name, feeds.url, users.name AS added_by
FROM feeds
LEFT JOIN users ON feeds.user_id = users.id;