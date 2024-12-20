package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	// "github.com/ansrivas/fiberprometheus/v2"
	"github.com/bytedance/sonic"
	"github.com/gofiber/contrib/swagger"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cache"
	"github.com/gofiber/fiber/v2/middleware/csrf"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/storage/postgres/v3"
	// "github.com/gofiber/storage/redis/v3"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/Trisamudrisvara/goTrip/db"
	"github.com/Trisamudrisvara/goTrip/routes"
)

func init() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Println("error loading .env file:", err)
	}
}

func main() {
	// Set SSL mode, default to "disable" if not specified
	sslmode := os.Getenv("SSLMODE")
	if sslmode == "" {
		sslmode = "disable"
	}

	dsn := os.Getenv("DB_URL")

	if dsn == "" {
		// Construct database connection string
		dsn = fmt.Sprintf(
			"host=%s user=%s password=%s dbname=%s sslmode=%s",
			os.Getenv("HOST"), os.Getenv("USER"),
			os.Getenv("PASS"), os.Getenv("NAME"), sslmode)

		// Add port to connection string if specified
		port := os.Getenv("PORT")
		if port != "" {
			dsn += " port=" + port
		}
	}

	// Create database connection
	ctx := context.Background()
	conn, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatal("error connecting db pool:", err)
	}
	defer conn.Close()

	// Initialize database queries and repository
	queries := db.New(conn)
	repo := &routes.Repo{
		Ctx:     ctx,
		Queries: queries,
	}

	// custom JSON encoder/decoder for performance
	fiberConfig := fiber.Config{
		// Prefork:     true,
		JSONEncoder: sonic.Marshal,
		JSONDecoder: sonic.Unmarshal,
	}

	// Initializing fiber app
	app := fiber.New(fiberConfig)

	// Configure CSRF middleware
	csrfConf := csrf.Config{
		KeyLookup:  "form:csrf",
		CookieName: "csrf",
		ContextKey: "csrf",
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			// log.Println("CSRF Error:", err)
			return c.Status(fiber.StatusForbidden).JSON(
				&fiber.Map{"error": "forbidden"})
		},
		Storage: postgres.New(postgres.Config{
			DB:    conn,
			Table: "csrf_token",
		})}

	// Configure Swagger
	swaggerConf := swagger.Config{
		Title:    "Trip API",
		FilePath: "swagger.yaml",
	}

	// Rate Limiter Config
	limiterConf := limiter.Config{
		Max:        3,
		Expiration: time.Second,
	}

	// Cache Config
	// cacheConf := cache.Config{
	// 	Storage:    redis.New(),
	// 	Expiration: 11 * time.Minute,
	// }

	// prometheus config
	// prometheus := fiberprometheus.New("trip")
	// prometheus.RegisterAt(app, "/metrics")
	// prometheus.SetSkipPaths([]string{"/ping", "/favicon.ico"})

	// Middlewares: logger, swagger, recover, cache, rate limiter & CSRF protection
	app.Use(logger.New(), limiter.New(limiterConf), cache.New(),
		swagger.New(swaggerConf), csrf.New(csrfConf), recover.New())

	// Set up routes
	repo.SetupRoutes(app)

	// Start the server
	port := os.Getenv("API_PORT")
	fmt.Println("Starting go server at port", port)
	port = ":" + port
	log.Fatal(app.Listen(port))
}
