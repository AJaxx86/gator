-- +goose up
CREATE TABLE posts(
	id UUID PRIMARY KEY,
	created_at TIME NOT NULL,
	updated_at TIME,
	title TEXT NOT NULL,
	url TEXT UNIQUE NOT NULL,
	description TEXT,
	published_at TIME,
	feed_id UUID REFERENCES feeds(id) ON DELETE CASCADE NOT NULL
);

-- +goose down
DROP TABLE posts;