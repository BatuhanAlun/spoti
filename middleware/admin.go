package middleware

import (
	"context"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
)

// DB ve Store global değişkenleri main.go'dan aktarılacak.
var DB *pgx.Conn

//var Store *session.Store

// AdminRequired, kullanıcının admin rolüne sahip olup olmadığını kontrol eden bir middleware'dır.
func AdminRequired(c *fiber.Ctx) error {
	sess, err := Store.Get(c)
	if err != nil {
		log.Printf("Session alınamadı: %v\n", err)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Oturum bulunamadı veya geçersiz."})
	}

	userIDLocal := sess.Get("userID")
	if userIDLocal == nil {
		// Eğer userID yoksa, kullanıcı oturum açmamıştır veya AuthRequired çalışmamıştır.
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Oturum açık değil."})
	}

	userID, ok := userIDLocal.(uuid.UUID)
	if !ok {
		log.Printf("AdminRequired: userID yerel değişkeni UUID tipinde değil: %v\n", userIDLocal)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Sunucu hatası, userID geçersiz."})
	}

	// Veritabanından kullanıcının rol adını (r.name) sorgula.
	var roleName string
	query := `
        SELECT r.name 
        FROM t_users u
        JOIN t_roles r ON u.role_id = r.id
        WHERE u.id = $1
    `
	err = DB.QueryRow(context.Background(), query, userID).Scan(&roleName)
	if err != nil {
		if err == pgx.ErrNoRows {
			// Kullanıcı bulunamazsa yetki reddedilir.
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Bu işlem için yetkiniz yok."})
		}
		log.Println("Admin yetkisi sorgulama hatası:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Yetki kontrolü sırasında bir hata oluştu."})
	}

	// Rol adının "admin" olup olmadığını kontrol et.
	if roleName != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Bu işlem için admin yetkisi gereklidir."})
	}

	// Eğer kullanıcı admin ise, bir sonraki işleyiciye geç.
	return c.Next()
}
