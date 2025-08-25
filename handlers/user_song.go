package handlers

import (
	"context"
	"log"
	"strconv"
	"strings"

	"spoti/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
)

// GetSongs, tüm şarkıları sayfalama ve arama filtreleriyle listeler.
func GetSongs(c *fiber.Ctx) error {
	// Sayfalama parametrelerini al
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit := 10
	offset := (page - 1) * limit

	// Arama parametresini al
	searchQuery := c.Query("search", "")

	var songs []models.Song
	var rows pgx.Rows
	var err error
	var count int

	// Sorgu ve arama için dinamik WHERE ve COUNT
	baseQuery := `SELECT id, title, artist, album, duration, click_count FROM t_songs`
	countQuery := `SELECT COUNT(*) FROM t_songs`
	whereClause := ""
	args := []interface{}{}

	if searchQuery != "" {
		// Arama sorgusunu küçük harfe çevir ve LIKE ile eşleştir
		whereClause = ` WHERE LOWER(title) LIKE $1`
		args = append(args, "%"+strings.ToLower(searchQuery)+"%")
	}

	// Toplam şarkı sayısını al
	err = DB.QueryRow(context.Background(), countQuery+whereClause, args...).Scan(&count)
	if err != nil {
		log.Println("Şarkı sayısı sorgu hatası:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Şarkılar listelenemedi."})
	}

	// Veritabanından şarkıları sırala, sayfala ve çek
	query := baseQuery + whereClause + ` ORDER BY click_count DESC LIMIT $` + strconv.Itoa(len(args)+1) + ` OFFSET $` + strconv.Itoa(len(args)+2)
	args = append(args, limit, offset)

	rows, err = DB.Query(context.Background(), query, args...)
	if err != nil {
		log.Println("Şarkı sorgulama hatası:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Şarkılar listelenemedi."})
	}
	defer rows.Close()

	for rows.Next() {
		var song models.Song
		err := rows.Scan(&song.ID, &song.Title, &song.Artist, &song.Album, &song.Duration, &song.ClickCount)
		if err != nil {
			log.Println("Şarkı tarama hatası:", err)
			continue
		}
		songs = append(songs, song)
	}

	return c.JSON(fiber.Map{
		"songs":     songs,
		"total":     count,
		"page":      page,
		"last_page": (count + limit - 1) / limit,
	})
}

// GetSongByID, bir şarkının detaylarını getirir ve tıklanma sayısını artırır.
func GetSongByID(c *fiber.Ctx) error {
	songID := c.Params("songID")
	parsedSongID, err := uuid.Parse(songID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Geçersiz şarkı ID'si."})
	}

	// Şarkının click_count değerini 1 artır
	_, err = DB.Exec(context.Background(), `UPDATE t_songs SET click_count = click_count + 1 WHERE id = $1`, parsedSongID)
	if err != nil {
		log.Println("Click count artırma hatası:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Şarkı güncellenemedi."})
	}

	var song models.Song
	query := `SELECT id, title, artist, album, duration, click_count FROM t_songs WHERE id = $1`
	err = DB.QueryRow(context.Background(), query, parsedSongID).Scan(&song.ID, &song.Title, &song.Artist, &song.Album, &song.Duration, &song.ClickCount)
	if err != nil {
		if err == pgx.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Şarkı bulunamadı."})
		}
		log.Println("Şarkı detay sorgu hatası:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Şarkı bilgileri alınamadı."})
	}

	return c.JSON(song)
}
