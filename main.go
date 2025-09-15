package main

import (
	"context"
	"encoding/gob"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/gofiber/storage/redis/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"

	"spoti/handlers"
	"spoti/middleware"
)

var store *session.Store
var db *pgx.Conn

func init() {
	// Gob paketine uuid.UUID tipini kaydet.
	// Bu, Fiber session modülünün UUID tipindeki verileri doğru şekilde saklamasını sağlar.
	gob.Register(uuid.UUID{})
}

func main() {
	// Redis için yeni bir Store oluştur

	redisStore := redis.New(redis.Config{
		Host: "redis",
		Port: 6379,
	})

	store = session.New(session.Config{
		Storage: redisStore,
	})

	// Veritabanı bağlantısı
	var err error
	connStr := "user=postgres password=postgres dbname=spoti host=db sslmode=disable"
	db, err = pgx.Connect(context.Background(), connStr)
	if err != nil {
		log.Fatalf("Veritabanına bağlanılamadı: %v\n", err)
	}
	defer db.Close(context.Background())

	err = db.Ping(context.Background())
	if err != nil {
		log.Fatalf("Veritabanı bağlantısı başarısız: %v\n", err)
	}

	app := fiber.New()

	// Handler'lara veritabanı ve session nesnelerini aktar
	handlers.DB = db
	handlers.Store = store
	middleware.Store = store
	middleware.DB = db

	api := app.Group("/api")

	// Kimlik Doğrulama (Authentication) rotaları
	api.Post("/user/register", handlers.RegisterUser)
	api.Post("/user/login", handlers.LoginUser)
	api.Post("/user/logout", handlers.LogoutUser)

	// --- API Route'ları ---
	// User API'leri için rotalar
	userAPI := api.Group("/user", middleware.AuthRequired)
	userAPI.Get("/", handlers.GetUser)
	userAPI.Put("/", handlers.UpdateUser)
	userAPI.Delete("/", handlers.DeleteUser)

	// Şarkı ve Çalma Listesi (Playlist) Rotaları
	userAPI.Get("/song", middleware.ValidatePageQuery, handlers.GetSongs)
	userAPI.Get("/song/:songID", handlers.GetSongByID)
	userAPI.Post("/playlist", handlers.CreatePlaylist)
	userAPI.Get("/playlist", handlers.GetUserPlaylists)
	userAPI.Get("/playlist/:playlistID", handlers.GetPlaylistByID)
	userAPI.Delete("/playlist/:playlistID", handlers.DeletePlaylist)
	userAPI.Post("/playlist/:playlistID/:songID", handlers.AddSongToPlaylist)

	// Rota çakışmasını önlemek için rotalar güncellendi
	userAPI.Get("/playlist/by-user/:userID", handlers.GetUserPlaylistsByUserID)
	userAPI.Get("/playlist/by-user/:userID/:songID", handlers.GetUserPlaylistSongByUserID)

	// Kupon ve Premium Üyelik Rotaları
	userAPI.Get("/coupon", handlers.GetUserCoupons)
	userAPI.Post("/premium/start", handlers.StartPremiumPurchase)
	userAPI.Post("/premium", handlers.PurchasePremium)

	// Admin API'leri için rotalar
	adminAPI := api.Group("/admin", middleware.AdminRequired)
	adminAPI.Get("/user", handlers.GetAllUsers)
	adminAPI.Get("/user/:userID", handlers.GetUserByID)
	adminAPI.Put("/user/:userID", handlers.UpdateUserByID)
	adminAPI.Delete("/user/:userID", handlers.DeleteUserByID)

	// Yeni Admin Rotaları
	adminAPI.Post("/song", handlers.AdminCreateSong)
	adminAPI.Delete("/song/:songID", handlers.AdminDeleteSong)
	adminAPI.Put("/song/:songID", handlers.AdminUpdateSong)

	// Kupon Admin Rotaları
	adminAPI.Post("/coupon", handlers.CreateCoupon)
	adminAPI.Post("/coupon/assign", handlers.AssignCoupon)

	log.Fatal(app.Listen(":3000"))
}
