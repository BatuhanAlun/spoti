package handlers

import (
	"context"
	"log"
	"spoti/models"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
)

// DB ve Store global değişkenleri main.go'dan aktarılacak
var DB *pgx.Conn
var Store *session.Store

// GetUser, oturumdaki kullanıcının bilgilerini getirir.
func GetUser(c *fiber.Ctx) error {
	userIDLocal := c.Locals("userID")
	if userIDLocal == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Oturum açık değil."})
	}

	userID, ok := userIDLocal.(uuid.UUID)
	if !ok {
		log.Printf("GetUser: userID yerel değişkeni UUID tipinde değil: %v\n", userIDLocal)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Sunucu hatası, userID geçersiz."})
	}

	var user models.User
	// Sorguya username ve email sütunları eklendi
	query := `SELECT id, username, email, role_id, hesap_turu, cash FROM t_users WHERE id = $1`
	err := DB.QueryRow(context.Background(), query, userID).Scan(&user.ID, &user.Username, &user.Email, &user.RoleID, &user.HesapTuru, &user.Cash)
	if err != nil {
		if err == pgx.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Kullanıcı bulunamadı."})
		}
		log.Println("Veritabanı sorgu hatası:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kullanıcı bilgileri alınamadı."})
	}

	return c.JSON(user)
}

// UpdateUser, oturumdaki kullanıcının bilgilerini günceller.
func UpdateUser(c *fiber.Ctx) error {
	userIDLocal := c.Locals("userID")
	if userIDLocal == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Oturum açık değil."})
	}

	userID, ok := userIDLocal.(uuid.UUID)
	if !ok {
		log.Printf("UpdateUser: userID yerel değişkeni UUID tipinde değil: %v\n", userIDLocal)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Sunucu hatası, userID geçersiz."})
	}

	var updatedUser models.User
	if err := c.BodyParser(&updatedUser); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Geçersiz istek gövdesi."})
	}

	query := `UPDATE t_users SET hesap_turu = $1, cash = $2 WHERE id = $3`
	commandTag, err := DB.Exec(context.Background(), query, updatedUser.HesapTuru, updatedUser.Cash, userID)
	if err != nil {
		log.Println("Veritabanı güncelleme hatası:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kullanıcı bilgileri güncellenemedi."})
	}

	if commandTag.RowsAffected() == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Güncellenecek kullanıcı bulunamadı."})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Kullanıcı başarıyla güncellendi."})
}

// DeleteUser, oturumdaki kullanıcının hesabını siler.
func DeleteUser(c *fiber.Ctx) error {
	userIDLocal := c.Locals("userID")
	if userIDLocal == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Oturum açık değil."})
	}

	userID, ok := userIDLocal.(uuid.UUID)
	if !ok {
		log.Printf("DeleteUser: userID yerel değişkeni UUID tipinde değil: %v\n", userIDLocal)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Sunucu hatası, userID geçersiz."})
	}

	query := `DELETE FROM t_users WHERE id = $1`
	commandTag, err := DB.Exec(context.Background(), query, userID)
	if err != nil {
		log.Println("Veritabanı silme hatası:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kullanıcı silinemedi."})
	}

	if commandTag.RowsAffected() == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Silinecek kullanıcı bulunamadı."})
	}

	sess, err := Store.Get(c)
	if err != nil {
		log.Println("Session alınamadı:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Oturum sonlandırılamadı."})
	}

	sess.Destroy()

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Kullanıcı başarıyla silindi."})
}
