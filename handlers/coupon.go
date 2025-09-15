package handlers

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"spoti/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
)

// Race condition'ları önlemek için global mutex
var mu sync.Mutex

// CreateCoupon, adminin yeni bir kupon oluşturmasını sağlar.
func CreateCoupon(c *fiber.Ctx) error {
	var coupon models.Coupon
	if err := c.BodyParser(&coupon); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Geçersiz istek gövdesi."})
	}

	// Kupon kodunun benzersiz olup olmadığını kontrol et
	var existingCouponID uuid.UUID
	err := DB.QueryRow(context.Background(), "SELECT id FROM t_cupons WHERE code = $1", coupon.Code).Scan(&existingCouponID)
	if err != nil && err != pgx.ErrNoRows {
		log.Println("Kupon kontrol hatası:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kupon oluşturulamadı."})
	}
	if err == nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": "Bu kupon kodu zaten mevcut."})
	}

	query := `INSERT INTO t_cupons (code) VALUES ($1) RETURNING id`
	err = DB.QueryRow(context.Background(), query, coupon.Code).Scan(&coupon.ID)
	if err != nil {
		log.Println("Kupon ekleme hatası:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kupon oluşturulamadı."})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "Kupon başarıyla oluşturuldu.", "couponID": coupon.ID})
}

// AssignCoupon, belirli bir kuponu belirli bir kullanıcıya veya tüm kullanıcılara atar.
func AssignCoupon(c *fiber.Ctx) error {
	var req struct {
		CouponID uuid.UUID `json:"cuponID"`
		UserID   uuid.UUID `json:"userID"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Geçersiz istek gövdesi."})
	}

	// Kuponun var olup olmadığını kontrol et
	var exists bool
	err := DB.QueryRow(context.Background(), "SELECT EXISTS(SELECT 1 FROM t_cupons WHERE id = $1)", req.CouponID).Scan(&exists)
	if err != nil {
		log.Println("Kupon kontrol hatası:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kupon atanamadı."})
	}
	if !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Kupon bulunamadı."})
	}

	// userID null uuid ise tüm kullanıcılara kupon ata
	if req.UserID == uuid.Nil {
		query := `UPDATE t_cupons SET user_id = u.id FROM t_users u WHERE t_cupons.id = $1`
		_, err := DB.Exec(context.Background(), query, req.CouponID)
		if err != nil {
			log.Println("Tüm kullanıcılara kupon atama hatası:", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kupon tüm kullanıcılara atanamadı."})
		}
		return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Kupon tüm kullanıcılara başarıyla atandı."})
	}

	// Belirli bir kullanıcıya kupon ata
	query := `UPDATE t_cupons SET user_id = $1 WHERE id = $2`
	commandTag, err := DB.Exec(context.Background(), query, req.UserID, req.CouponID)
	if err != nil {
		log.Println("Kullanıcıya kupon atama hatası:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kupon kullanıcıya atanamadı."})
	}

	if commandTag.RowsAffected() == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Kupon veya kullanıcı bulunamadı."})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Kupon kullanıcıya başarıyla atandı."})
}

// GetUserCoupons, kullanıcının sahip olduğu kuponları listeler.
func GetUserCoupons(c *fiber.Ctx) error {
	sess, err := Store.Get(c)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Session alınamadı."})
	}

	userID := sess.Get("userID")
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Yetkilendirme hatası: Kullanıcı ID'si session'da bulunamadı."})
	}

	var coupons []models.Coupon
	rows, err := DB.Query(context.Background(), `SELECT id, code, is_used FROM t_cupons WHERE user_id = $1`, userID)
	if err != nil {
		log.Println("Kuponları sorgulama hatası:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Kuponlar listelenemedi."})
	}
	defer rows.Close()

	for rows.Next() {
		var coupon models.Coupon
		if err := rows.Scan(&coupon.ID, &coupon.Code, &coupon.IsUsed); err != nil {
			log.Println("Satır tarama hatası:", err)
			continue
		}
		coupons = append(coupons, coupon)
	}

	return c.Status(fiber.StatusOK).JSON(coupons)
}

// StartPremiumPurchase, kullanıcının premium satın alma işlemini başlatır ve session'a bir zaman damgası ekler.
func StartPremiumPurchase(c *fiber.Ctx) error {
	sess, err := Store.Get(c)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Session alınamadı."})
	}

	// Session'da userID kontrolü
	if sess.Get("userID") == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Yetkilendirme hatası: Kullanıcı ID'si session'da bulunamadı."})
	}

	// Satın alma oturumu için son kullanma zamanı belirle (örneğin 5 dakika)
	purchaseExpirationTime := time.Now().Add(5 * time.Minute)
	sess.Set("purchaseExpirationTime", purchaseExpirationTime)

	if err := sess.Save(); err != nil {
		fmt.Println(err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Session kaydedilemedi."})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Premium satın alma işlemi başlatıldı. İşleminizi tamamlamak için 5 dakikanız var."})
}

// PurchasePremium, kullanıcının premium üyelik satın almasını sağlar.
func PurchasePremium(c *fiber.Ctx) error {
	// Mutex kilidi kullanarak race condition'ı önle
	mu.Lock()
	defer mu.Unlock()

	var req struct {
		UseCoupon bool   `json:"useCoupon"`
		CouponID  string `json:"couponID"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Geçersiz istek gövdesi."})
	}

	sess, err := Store.Get(c)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Session alınamadı."})
	}

	// Session'da userID kontrolü
	userID := sess.Get("userID")
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Yetkilendirme hatası: Kullanıcı ID'si session'da bulunamadı."})
	}

	// Kullanıcının mevcut hesap türünü kontrol et
	var currentUser models.User
	err = DB.QueryRow(c.Context(), "SELECT hesap_turu FROM t_users WHERE id = $1", userID).Scan(&currentUser.HesapTuru)
	if err == pgx.ErrNoRows {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Kullanıcı bulunamadı."})
	}
	if err != nil {
		log.Println("Hesap türü sorgulama hatası:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "İşlem sırasında bir hata oluştu."})
	}
	if currentUser.HesapTuru == "Premium" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Kullanıcı zaten premium üyedir."})
	}

	// Satın alma oturumu zaman aşımı kontrolü
	expirationTime, ok := sess.Get("purchaseExpirationTime").(time.Time)
	if !ok || time.Now().After(expirationTime) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Satın alma oturumu geçerli değil veya süresi dolmuş. Lütfen işlemi yeniden başlatın."})
	}

	// Timeout için select ifadesini kullan
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if req.UseCoupon {
		// Kupon kullanarak satın alma
		var couponID uuid.UUID
		if req.CouponID != "" {
			couponID, err = uuid.Parse(req.CouponID)
			if err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Geçersiz kupon ID formatı."})
			}
		} else {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Kupon ID'si sağlanmadı."})
		}

		// Kuponun geçerliliğini ve kullanım durumunu kontrol et
		var coupon models.Coupon
		err = DB.QueryRow(ctx, "SELECT id, is_used FROM t_cupons WHERE id = $1 AND user_id = $2", couponID, userID).Scan(&coupon.ID, &coupon.IsUsed)
		if err == pgx.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Kupon bulunamadı veya bu kupon size ait değil."})
		}
		if err != nil {
			log.Println("Kupon kontrol hatası:", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "İşlem sırasında bir hata oluştu."})
		}
		if coupon.IsUsed {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Bu kupon zaten kullanılmış."})
		}

		// Veritabanı işlemleri için select/timeout
		select {
		case <-ctx.Done():
			return c.Status(fiber.StatusGatewayTimeout).JSON(fiber.Map{"error": "İşlem zaman aşımına uğradı."})
		default:
			tx, err := DB.Begin(ctx)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "İşlem başlatılamadı."})
			}
			defer tx.Rollback(ctx)

			// Kuponu kullanıldı olarak işaretle
			if _, err := tx.Exec(ctx, "UPDATE t_cupons SET is_used = TRUE WHERE id = $1", couponID); err != nil {
				log.Println("Kupon güncelleme hatası:", err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "İşlem sırasında bir hata oluştu."})
			}

			// Kullanıcının hesap türünü Premium yap
			if _, err := tx.Exec(ctx, "UPDATE t_users SET hesap_turu = 'Premium' WHERE id = $1", userID); err != nil {
				log.Println("Kullanıcı güncelleme hatası:", err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "İşlem sırasında bir hata oluştu."})
			}

			if err := tx.Commit(ctx); err != nil {
				log.Println("İşlem taahhüt hatası:", err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "İşlem tamamlanamadı."})
			}

			// Oturumu temizle
			sess.Delete("purchaseExpirationTime")
			sess.Save()

			return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Premium üyelik kupon ile başarıyla etkinleştirildi."})
		}
	} else {
		// Nakit kullanarak satın alma
		var user models.User
		err = DB.QueryRow(ctx, "SELECT cash FROM t_users WHERE id = $1", userID).Scan(&user.Cash)
		if err == pgx.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Kullanıcı bulunamadı."})
		}
		if err != nil {
			log.Println("Kullanıcı nakit sorgulama hatası:", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "İşlem sırasında bir hata oluştu."})
		}

		const price = 100.0

		if user.Cash < price {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Yetersiz bakiye."})
		}

		// Veritabanı işlemleri için select/timeout
		select {
		case <-ctx.Done():
			return c.Status(fiber.StatusGatewayTimeout).JSON(fiber.Map{"error": "İşlem zaman aşımına uğradı."})
		default:
			tx, err := DB.Begin(ctx)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "İşlem başlatılamadı."})
			}
			defer tx.Rollback(ctx)

			// Kullanıcının bakiyesini güncelle
			newCash := user.Cash - price
			if _, err := tx.Exec(ctx, "UPDATE t_users SET cash = $1, hesap_turu = 'Premium' WHERE id = $2", newCash, userID); err != nil {
				log.Println("Nakit güncelleme hatası:", err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "İşlem sırasında bir hata oluştu."})
			}

			if err := tx.Commit(ctx); err != nil {
				log.Println("İşlem taahhüt hatası:", err)
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "İşlem tamamlanamadı."})
			}

			// Oturumu temizle
			sess.Delete("purchaseExpirationTime")
			sess.Save()

			return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Premium üyelik başarıyla satın alındı.", "newCash": newCash})
		}
	}
}
