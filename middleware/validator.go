package middleware

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
)

// ValidatePageQuery, isteğin "page" sorgu parametresini kontrol eder.
// Eğer "page" parametresi geçerli değilse, 400 Bad Request hatası döndürür.
func ValidatePageQuery(c *fiber.Ctx) error {
	pageStr := c.Query("page", "1")
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		// Geçersiz bir değerse 400 hatası döndür
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid page number",
			"message": "Sayfa numarası pozitif bir tam sayı olmalıdır.",
		})
	}
	// Eğer geçerliyse, sonraki middleware veya handler'a geç
	return c.Next()
}
