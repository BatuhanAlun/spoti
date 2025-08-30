package main

import (
	"context"
	"encoding/gob"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
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
	store = session.New()

	// Veritabanı bağlantısı
	var err error
	connStr := "user=postgres password=postgres dbname=spoti host=localhost sslmode=disable"
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

	// Kimlik Doğrulama (Authentication) rotaları
	app.Post("/user/register", handlers.RegisterUser)
	app.Post("/user/login", handlers.LoginUser)
	app.Post("/user/logout", handlers.LogoutUser)

	// --- API Route'ları ---
	// Bu rotalar artık AuthRequired middleware'ı ile korunuyor
	userAPI := app.Group("/user", middleware.AuthRequired)
	userAPI.Get("/", handlers.GetUser)
	userAPI.Put("/", handlers.UpdateUser)
	userAPI.Delete("/", handlers.DeleteUser)

	// TASK 9: Şarkı ve Çalma Listesi (Playlist) Rotaları
	userAPI.Get("/song", middleware.ValidatePageQuery, handlers.GetSongs)
	userAPI.Get("/song/:songID", handlers.GetSongByID)
	userAPI.Post("/playlist", handlers.CreatePlaylist)
	userAPI.Get("/playlist", handlers.GetUserPlaylists)
	userAPI.Get("/playlist/:playlistID", handlers.GetPlaylistByID)
	userAPI.Delete("/playlist/:playlistID", handlers.DeletePlaylist)
	userAPI.Post("/playlist/:playlistID/:songID", handlers.AddSongToPlaylist)
	userAPI.Get("/playlist/:userID", handlers.GetUserPlaylistsByUserID)
	userAPI.Get("/playlist/:userID/:songID", handlers.GetUserPlaylistSongByUserID)

	// Admin API'leri için rotalar
	adminAPI := app.Group("/admin", middleware.AdminRequired)
	adminAPI.Get("/user", handlers.GetAllUsers)
	adminAPI.Get("/user/:userID", handlers.GetUserByID)
	adminAPI.Put("/user/:userID", handlers.UpdateUserByID)
	adminAPI.Delete("/user/:userID", handlers.DeleteUserByID)

	// TASK 9: Yeni Admin Rotaları
	adminAPI.Post("/song", handlers.AdminCreateSong)
	adminAPI.Delete("/song/:songID", handlers.AdminDeleteSong)
	adminAPI.Put("/song/:songID", handlers.AdminUpdateSong)

	log.Fatal(app.Listen(":3000"))
}
