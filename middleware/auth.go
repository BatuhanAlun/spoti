package middleware

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/google/uuid"
)

// Store değişkeni, main.go dosyasında atanacaktır.
var Store *session.Store

// AuthRequired middleware'ı, sadece oturum açmış kullanıcıların erişimine izin verir.
func AuthRequired(c *fiber.Ctx) error {
	if Store == nil {
		log.Println("Session store atanamadı. Bu bir konfigürasyon hatasıdır.")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Sunucu hatası, session store mevcut değil."})
	}

	// Oturumu al
	sess, err := Store.Get(c)
	if err != nil {
		log.Printf("Session alınamadı: %v\n", err)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Oturum bulunamadı veya geçersiz."})
	}

	// Session'da userID olup olmadığını kontrol et
	userID := sess.Get("userID")
	if userID == nil {
		log.Println("userID bulunamadı, oturum yetkisiz.")
		_ = sess.Destroy()
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Lütfen giriş yapın."})
	}

	// userID'yi bir sonraki handler'a iletmek için Local değişkenine kaydet
	// Bu, değeri uuid.UUID tipinde tutar.
	c.Locals("userID", userID.(uuid.UUID))

	// İstek zincirinde bir sonraki middleware veya handlera geç
	return c.Next()
}
