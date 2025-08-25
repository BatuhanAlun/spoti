-- +goose Up
CREATE TABLE t_songs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title VARCHAR(255) NOT NULL,
    artist VARCHAR(255) NOT NULL,
    album VARCHAR(255),
    duration INT,
    click_count INT DEFAULT 0
);

CREATE TABLE t_playlist (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    user_id UUID NOT NULL REFERENCES t_users(id) ON DELETE CASCADE
);

CREATE TABLE t_playlist_songs (
    playlist_id UUID REFERENCES t_playlist(id) ON DELETE CASCADE,
    song_id UUID REFERENCES t_songs(id) ON DELETE CASCADE,
    PRIMARY KEY (playlist_id, song_id)
);

-- +goose Down
DROP TABLE t_playlist_songs;
DROP TABLE t_playlist;
DROP TABLE t_songs;