package handlers

import (
	"context"
	"log"

	"spoti/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
)

// GetAllUsers, veritabanındaki tüm kullanıcıları listeler.
func GetAllUsers(c *fiber.Ctx) error {
	var users []models.User
	// Sorguya username ve email sütunları eklendi.
	rows, err := DB.Query(context.Background(), `SELECT id, username, email, role_id, hesap_turu, cash FROM t_users`)
	if err != nil {
		log.Println("Tüm kullanıcıları sorgulama hatası:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kullanıcılar listelenemedi."})
	}
	defer rows.Close()

	for rows.Next() {
		var user models.User
		// Scan fonksiyonu, sorgudaki yeni sütunları içerecek şekilde güncellendi.
		if err := rows.Scan(&user.ID, &user.Username, &user.Email, &user.RoleID, &user.HesapTuru, &user.Cash); err != nil {
			log.Println("Satır tarama hatası:", err)
			continue
		}
		users = append(users, user)
	}

	return c.JSON(users)
}

// GetUserByID, belirli bir kullanıcıyı ID'sine göre getirir (Admin yetkilendirmesi gereklidir).
func GetUserByID(c *fiber.Ctx) error {
	userID := c.Params("userID")
	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Geçersiz kullanıcı ID'si."})
	}

	var user models.User
	// Sorguya username ve email sütunları eklendi.
	query := `SELECT id, username, email, role_id, hesap_turu, cash FROM t_users WHERE id = $1`
	// Scan fonksiyonu, sorgudaki yeni sütunları içerecek şekilde güncellendi.
	err = DB.QueryRow(context.Background(), query, parsedUserID).Scan(&user.ID, &user.Username, &user.Email, &user.RoleID, &user.HesapTuru, &user.Cash)
	if err != nil {
		if err == pgx.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Kullanıcı bulunamadı."})
		}
		log.Println("Kullanıcı ID ile sorgu hatası:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kullanıcı bilgileri alınamadı."})
	}

	return c.JSON(user)
}

// UpdateUserByID, belirli bir kullanıcının bilgilerini günceller.
func UpdateUserByID(c *fiber.Ctx) error {
	userID := c.Params("userID")
	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Geçersiz kullanıcı ID'si."})
	}

	var updatedUser models.User
	if err := c.BodyParser(&updatedUser); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Geçersiz istek gövdesi."})
	}

	query := `UPDATE t_users SET hesap_turu = $1, cash = $2 WHERE id = $3`
	commandTag, err := DB.Exec(context.Background(), query, updatedUser.HesapTuru, updatedUser.Cash, parsedUserID)
	if err != nil {
		log.Println("Veritabanı güncelleme hatası:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kullanıcı bilgileri güncellenemedi."})
	}

	if commandTag.RowsAffected() == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Güncellenecek kullanıcı bulunamadı."})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Kullanıcı başarıyla güncellendi."})
}

// DeleteUserByID, belirli bir kullanıcıyı ID'sine göre siler.
func DeleteUserByID(c *fiber.Ctx) error {
	userID := c.Params("userID")
	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Geçersiz kullanıcı ID'si."})
	}

	query := `DELETE FROM t_users WHERE id = $1`
	commandTag, err := DB.Exec(context.Background(), query, parsedUserID)
	if err != nil {
		log.Println("Veritabanı silme hatası:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kullanıcı silinemedi."})
	}

	if commandTag.RowsAffected() == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Silinecek kullanıcı bulunamadı."})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Kullanıcı başarıyla silindi."})
}
