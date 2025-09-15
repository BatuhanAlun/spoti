package handlers

import (
	"context"
	"log"

	"spoti/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
	"golang.org/x/crypto/bcrypt"
)

// RegisterUser, yeni bir kullanıcı kaydı oluşturur.
func RegisterUser(c *fiber.Ctx) error {
	var user models.User
	if err := c.BodyParser(&user); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Geçersiz istek gövdesi."})
	}

	var userRoleID uuid.UUID
	err := DB.QueryRow(context.Background(), "SELECT id FROM t_roles WHERE name = 'user'").Scan(&userRoleID)
	if err == pgx.ErrNoRows {

		log.Println("Veritabanında 'user' rolü bulunamadı.")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "'user' rolü veritabanında bulunamadı. Lütfen yöneticiyle iletişime geçin."})
	}
	if err != nil {
		log.Printf("'user' rolü sorgulama hatası: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Rol ataması sırasında bir hata oluştu."})
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Şifre hashleme hatası: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Şifre hashleme hatası."})
	}
	user.Password = string(hashedPassword)
	user.ID = uuid.New()
	user.HesapTuru = "Free"
	user.Cash = 100.00
	user.RoleID = userRoleID

	query := `INSERT INTO t_users (id, username, email, password, hesap_turu, cash, role_id) VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err = DB.Exec(context.Background(), query, user.ID, user.Username, user.Email, user.Password, user.HesapTuru, user.Cash, user.RoleID)
	if err != nil {
		log.Printf("Kullanıcı kaydı oluşturma hatası: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kullanıcı kaydı yapılamadı."})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "Kullanıcı başarıyla kaydedildi.", "userID": user.ID})
}

// LoginUser, kullanıcının oturum açmasını sağlar.
func LoginUser(c *fiber.Ctx) error {
	var loginData struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.BodyParser(&loginData); err != nil {
		log.Printf("Giriş verilerini çözümleme hatası: %v\n", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Geçersiz istek gövdesi."})
	}

	var user models.User
	query := `SELECT id, password FROM t_users WHERE email = $1`
	err := DB.QueryRow(context.Background(), query, loginData.Email).Scan(&user.ID, &user.Password)
	if err != nil {
		log.Printf("Kullanıcı bulunamadı veya veritabanı hatası: %v\n", err)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Geçersiz e-posta veya şifre."})
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(loginData.Password)); err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Geçersiz e-posta veya şifre."})
	}

	// Oturum oluştur ve kullanıcı ID'sini sakla
	sess, err := Store.Get(c)
	if err != nil {
		log.Printf("Session oluşturma/alma hatası: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Session oluşturulamadı."})
	}
	sess.Set("userID", user.ID)
	if err := sess.Save(); err != nil {
		log.Printf("Session kaydetme hatası: %v\n", err) // Hata kaynağını daha iyi anlamak için log ekledik.
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Session kaydedilemedi."})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Giriş başarılı."})
}

// LogoutUser, kullanıcının oturumunu kapatır.
func LogoutUser(c *fiber.Ctx) error {
	sess, err := Store.Get(c)
	if err != nil {
		log.Printf("Session alma hatası: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Session alınamadı."})
	}
	if err := sess.Destroy(); err != nil {
		log.Printf("Session sonlandırma hatası: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Session sonlandırılamadı."})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Çıkış başarılı."})
}
