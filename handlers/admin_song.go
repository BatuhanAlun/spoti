package handlers

import (
	"context"
	"log"

	"spoti/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// Admin, yeni bir şarkı ekler.
func AdminCreateSong(c *fiber.Ctx) error {
	var song models.Song
	if err := c.BodyParser(&song); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Geçersiz istek gövdesi."})
	}

	// Veritabanına yeni şarkıyı ekle
	query := `INSERT INTO t_songs (title, artist, album, duration) VALUES ($1, $2, $3, $4) RETURNING id`
	err := DB.QueryRow(context.Background(), query, song.Title, song.Artist, song.Album, song.Duration).Scan(&song.ID)
	if err != nil {
		log.Println("Şarkı ekleme hatası:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Şarkı eklenemedi."})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "Şarkı başarıyla eklendi.", "song_id": song.ID})
}

// Admin, bir şarkıyı siler.
func AdminDeleteSong(c *fiber.Ctx) error {
	songID := c.Params("songID")
	parsedSongID, err := uuid.Parse(songID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Geçersiz şarkı ID'si."})
	}

	commandTag, err := DB.Exec(context.Background(), `DELETE FROM t_songs WHERE id = $1`, parsedSongID)
	if err != nil {
		log.Println("Şarkı silme hatası:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Şarkı silinemedi."})
	}

	if commandTag.RowsAffected() == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Silinecek şarkı bulunamadı."})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Şarkı başarıyla silindi."})
}

// Admin, bir şarkının bilgilerini günceller.
func AdminUpdateSong(c *fiber.Ctx) error {
	songID := c.Params("songID")
	parsedSongID, err := uuid.Parse(songID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Geçersiz şarkı ID'si."})
	}

	var updatedSong models.Song
	if err := c.BodyParser(&updatedSong); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Geçersiz istek gövdesi."})
	}

	// Veritabanında güncelleme yap
	query := `UPDATE t_songs SET title = $1, artist = $2, album = $3, duration = $4 WHERE id = $5`
	commandTag, err := DB.Exec(context.Background(), query, updatedSong.Title, updatedSong.Artist, updatedSong.Album, updatedSong.Duration, parsedSongID)
	if err != nil {
		log.Println("Şarkı güncelleme hatası:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Şarkı güncellenemedi."})
	}

	if commandTag.RowsAffected() == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Güncellenecek şarkı bulunamadı."})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Şarkı başarıyla güncellendi."})
}
