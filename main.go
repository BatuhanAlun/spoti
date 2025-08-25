package main

import (
	"context"
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/jackc/pgx/v4"

	"spoti/handlers"
	// Yeni middleware paketimizi dahil et
)

var store *session.Store
var db *pgx.Conn

func main() {
	store = session.New()

	// Veritabanı bağlantısı
	var err error
	connStr := "user=youruser password=yourpassword dbname=yourdb host=localhost sslmode=disable"
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

	// --- API Route'ları ---
	userAPI := app.Group("/user")
	userAPI.Get("/", handlers.GetUser)
	userAPI.Put("/", handlers.UpdateUser)
	userAPI.Delete("/", handlers.DeleteUser)

	adminAPI := app.Group("/admin")
	adminAPI.Get("/user", handlers.GetAllUsers)
	adminAPI.Get("/user/:userID", handlers.GetUserByID)
	adminAPI.Put("/user/:userID", handlers.UpdateUserByID)
	adminAPI.Delete("/user/:userID", handlers.DeleteUserByID)

	// Örnek: TASK 9 için sayfalama middleware'ını entegre etme
	// Bu satırı TASK 9'da yapacağın /user/song rotası için kullanacaksın.
	// app.Get("/user/song", middleware.ValidatePageQuery, handlers.GetSongs)

	log.Fatal(app.Listen(":3000"))
}
