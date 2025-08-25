package handlers

import (
	"context"
	"log"

	"spoti/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
)

// Global değişkenler main.go'dan atanacaktır.
//var DB *pgx.Conn
//var Store *session.Store

// CreatePlaylist, kullanıcının yeni bir çalma listesi oluşturmasını sağlar.
func CreatePlaylist(c *fiber.Ctx) error {
	// Middleware'dan userID'yi al.
	userIDLocal := c.Locals("userID")
	if userIDLocal == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Oturum açık değil."})
	}
	userID, ok := userIDLocal.(uuid.UUID)
	if !ok {
		log.Printf("CreatePlaylist: userID yerel değişkeni UUID tipinde değil: %v\n", userIDLocal)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Sunucu hatası, userID geçersiz."})
	}

	var playlist models.Playlist
	if err := c.BodyParser(&playlist); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Geçersiz istek gövdesi."})
	}
	playlist.UserID = userID

	query := `INSERT INTO t_playlist (name, user_id) VALUES ($1, $2) RETURNING id`
	err := DB.QueryRow(context.Background(), query, playlist.Name, playlist.UserID).Scan(&playlist.ID)
	if err != nil {
		log.Println("Playlist oluşturma hatası:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Çalma listesi oluşturulamadı."})
	}

	return c.Status(fiber.StatusCreated).JSON(playlist)
}

// GetUserPlaylists, oturumdaki kullanıcının çalma listelerini getirir.
func GetUserPlaylists(c *fiber.Ctx) error {
	// Middleware'dan userID'yi al.
	userIDLocal := c.Locals("userID")
	if userIDLocal == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Oturum açık değil."})
	}
	userID, ok := userIDLocal.(uuid.UUID)
	if !ok {
		log.Printf("GetUserPlaylists: userID yerel değişkeni UUID tipinde değil: %v\n", userIDLocal)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Sunucu hatası, userID geçersiz."})
	}

	var playlists []models.Playlist
	rows, err := DB.Query(context.Background(), `SELECT id, name, user_id FROM t_playlist WHERE user_id = $1`, userID)
	if err != nil {
		log.Println("Kullanıcı çalma listeleri sorgulama hatası:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Çalma listeleri alınamadı."})
	}
	defer rows.Close()

	for rows.Next() {
		var playlist models.Playlist
		if err := rows.Scan(&playlist.ID, &playlist.Name, &playlist.UserID); err != nil {
			log.Println("Playlist satır tarama hatası:", err)
			continue
		}
		playlists = append(playlists, playlist)
	}

	// Rows döngüsünden sonra olası hataları kontrol et.
	if err := rows.Err(); err != nil {
		log.Printf("Döngü sonrası hata: %v\n", err)
	}

	return c.JSON(playlists)
}

// GetPlaylistByID, belirli bir çalma listesini ve içindeki şarkıları getirir.
func GetPlaylistByID(c *fiber.Ctx) error {
	playlistID := c.Params("playlistID")
	parsedPlaylistID, err := uuid.Parse(playlistID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Geçersiz çalma listesi ID'si."})
	}

	var playlist models.Playlist
	query := `SELECT id, name, user_id FROM t_playlist WHERE id = $1`
	err = DB.QueryRow(context.Background(), query, parsedPlaylistID).Scan(&playlist.ID, &playlist.Name, &playlist.UserID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Çalma listesi bulunamadı."})
		}
		log.Println("Playlist sorgu hatası:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Çalma listesi alınamadı."})
	}

	var songs []models.Song
	songsQuery := `
        SELECT s.id, s.title, s.artist, s.album, s.duration, s.click_count 
        FROM t_playlist_songs ps
        JOIN t_songs s ON ps.song_id = s.id
        WHERE ps.playlist_id = $1
    `
	rows, err := DB.Query(context.Background(), songsQuery, parsedPlaylistID)
	if err != nil {
		log.Println("Playlist şarkıları sorgulama hatası:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Çalma listesi şarkıları alınamadı."})
	}
	defer rows.Close()

	for rows.Next() {
		var song models.Song
		if err := rows.Scan(&song.ID, &song.Title, &song.Artist, &song.Album, &song.Duration, &song.ClickCount); err != nil {
			log.Println("Şarkı satır tarama hatası:", err)
			continue
		}
		songs = append(songs, song)
	}
	// rows.Next() döngüsünden sonra olası hataları kontrol et.
	if err := rows.Err(); err != nil {
		log.Printf("Döngü sonrası hata: %v\n", err)
	}

	return c.JSON(fiber.Map{"playlist": playlist, "songs": songs})
}

// DeletePlaylist, belirli bir çalma listesini siler.
func DeletePlaylist(c *fiber.Ctx) error {
	playlistID := c.Params("playlistID")
	parsedPlaylistID, err := uuid.Parse(playlistID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Geçersiz çalma listesi ID'si."})
	}

	// Middleware'dan userID'yi al.
	userIDLocal := c.Locals("userID")
	if userIDLocal == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Oturum açık değil."})
	}
	userID, ok := userIDLocal.(uuid.UUID)
	if !ok {
		log.Printf("DeletePlaylist: userID yerel değişkeni UUID tipinde değil: %v\n", userIDLocal)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Sunucu hatası, userID geçersiz."})
	}

	// Kullanıcının kendi playlist'ini silmeye yetkisi var mı kontrol et
	var ownerID uuid.UUID
	err = DB.QueryRow(context.Background(), `SELECT user_id FROM t_playlist WHERE id = $1`, parsedPlaylistID).Scan(&ownerID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Çalma listesi bulunamadı."})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Veritabanı hatası."})
	}

	if ownerID != userID {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Bu çalma listesini silmeye yetkiniz yok."})
	}

	commandTag, err := DB.Exec(context.Background(), `DELETE FROM t_playlist WHERE id = $1`, parsedPlaylistID)
	if err != nil {
		log.Println("Playlist silme hatası:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Çalma listesi silinemedi."})
	}

	if commandTag.RowsAffected() == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Silinecek çalma listesi bulunamadı."})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Çalma listesi başarıyla silindi."})
}

// AddSongToPlaylist, bir şarkıyı çalma listesine ekler. Free kullanıcılar için 5 şarkı sınırı vardır.
func AddSongToPlaylist(c *fiber.Ctx) error {
	playlistID := c.Params("playlistID")
	songID := c.Params("songID")
	parsedPlaylistID, err := uuid.Parse(playlistID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Geçersiz çalma listesi ID'si."})
	}
	parsedSongID, err := uuid.Parse(songID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Geçersiz şarkı ID'si."})
	}

	// Middleware'dan userID'yi al.
	userIDLocal := c.Locals("userID")
	if userIDLocal == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Oturum açık değil."})
	}

	// Kullanıcının hesap türünü kontrol et
	var accountType string
	err = DB.QueryRow(context.Background(), `SELECT tu.hesap_turu FROM t_users tu JOIN t_playlist tp ON tu.id = tp.user_id WHERE tp.id = $1`, parsedPlaylistID).Scan(&accountType)
	if err != nil {
		if err == pgx.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Çalma listesi bulunamadı."})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kullanıcı bilgisi alınamadı."})
	}

	if accountType == "Free" {
		// Free kullanıcı için şarkı sayısını kontrol et
		var songCount int
		err = DB.QueryRow(context.Background(), `SELECT COUNT(*) FROM t_playlist_songs WHERE playlist_id = $1`, parsedPlaylistID).Scan(&songCount)
		if err != nil {
			log.Println("Şarkı sayısı sorgu hatası:", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Şarkı sayısı kontrol edilemedi."})
		}
		if songCount >= 5 {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Ücretsiz kullanıcılar bir çalma listesine en fazla 5 şarkı ekleyebilir."})
		}
	}

	// Çalma listesine şarkıyı ekle
	query := `INSERT INTO t_playlist_songs (playlist_id, song_id) VALUES ($1, $2)`
	_, err = DB.Exec(context.Background(), query, parsedPlaylistID, parsedSongID)
	if err != nil {
		log.Println("Şarkı ekleme hatası:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Şarkı çalma listesine eklenemedi."})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Şarkı başarıyla çalma listesine eklendi."})
}

// GetUserPlaylistsByUserID, belirli bir kullanıcının tüm çalma listelerini getirir.
func GetUserPlaylistsByUserID(c *fiber.Ctx) error {
	parsedUserID, err := uuid.Parse(c.Params("userID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Geçersiz kullanıcı ID'si."})
	}

	var playlists []models.Playlist
	rows, err := DB.Query(context.Background(), `SELECT id, name, user_id FROM t_playlist WHERE user_id = $1`, parsedUserID)
	if err != nil {
		log.Println("Kullanıcı çalma listeleri sorgulama hatası:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Çalma listeleri alınamadı."})
	}
	defer rows.Close()

	for rows.Next() {
		var playlist models.Playlist
		if err := rows.Scan(&playlist.ID, &playlist.Name, &playlist.UserID); err != nil {
			log.Println("Playlist satır tarama hatası:", err)
			continue
		}
		playlists = append(playlists, playlist)
	}

	// rows.Next() döngüsünden sonra olası hataları kontrol et.
	if err := rows.Err(); err != nil {
		log.Printf("Döngü sonrası hata: %v\n", err)
	}

	return c.JSON(playlists)
}

// GetUserPlaylistSongByUserID, belirli bir kullanıcının çalma listesindeki belirli bir şarkıyı getirir.
func GetUserPlaylistSongByUserID(c *fiber.Ctx) error {
	parsedUserID, err := uuid.Parse(c.Params("userID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Geçersiz kullanıcı ID'si."})
	}
	parsedSongID, err := uuid.Parse(c.Params("songID"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Geçersiz şarkı ID'si."})
	}

	var song models.Song
	query := `
        SELECT s.id, s.title, s.artist, s.album, s.duration, s.click_count 
        FROM t_playlist_songs ps
        JOIN t_playlist p ON ps.playlist_id = p.id
        JOIN t_songs s ON ps.song_id = s.id
        WHERE p.user_id = $1 AND s.id = $2
    `
	err = DB.QueryRow(context.Background(), query, parsedUserID, parsedSongID).Scan(&song.ID, &song.Title, &song.Artist, &song.Album, &song.Duration, &song.ClickCount)
	if err != nil {
		if err == pgx.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Şarkı bu kullanıcının çalma listesinde bulunamadı."})
		}
		log.Println("Şarkı sorgu hatası:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Şarkı bilgileri alınamadı."})
	}

	return c.JSON(song)
}
