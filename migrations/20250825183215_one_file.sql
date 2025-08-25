-- +goose Up
-- Bu migration, t_roles ve t_users tablolarını oluşturur.
-- t_users tablosu, username, email ve password sütunlarını içerir.

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- t_roles tablosu, kullanıcı rollerini (örneğin, Admin, User) saklar.
CREATE TABLE IF NOT EXISTS t_roles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4()
);

-- t_users tablosu, kullanıcı bilgilerini saklar.
CREATE TABLE IF NOT EXISTS t_users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    role_id UUID REFERENCES t_roles(id),
    username VARCHAR(50) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    password TEXT NOT NULL,
    hesap_turu VARCHAR(50) DEFAULT 'Free',
    cash DECIMAL(10, 2) DEFAULT 100.00,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

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
-- Bu migration, ilgili tabloları doğru sırayla siler.
-- Bağımlılıkları olan tablolar önce silinmelidir.

DROP TABLE IF EXISTS t_playlist_songs;
DROP TABLE IF EXISTS t_playlist;
DROP TABLE IF EXISTS t_songs;
DROP TABLE IF EXISTS t_users;
DROP TABLE IF EXISTS t_roles;
