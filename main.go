package main

import (
	"context"
	"fmt"
	"log"
	"scanner/config"
	"scanner/databases"
	"scanner/internal/middlewares"
	"scanner/internal/routes"
	"scanner/internal/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/google/uuid"
)

func main() {
	fmt.Println(uuid.New().String())
	config := config.InitConfig()
	databases.InitialMongoDB(config)
	defer databases.CloseMongoDB()
	app := fiber.New(fiber.Config{
		ProxyHeader: "X-Forwarded-For",
		BodyLimit:   200 * 1024 * 1024, // 100 MB for large file uploads
	})

	// Initialize OIDC provider and verifier
	if err := utils.InitOIDCProvider(context.Background(), config.OIDCProvider.Authority, config.OIDCProvider.ExpectedAudience, config.OIDCProvider.RequiredScopes); err != nil {
		log.Fatalf("Failed to initialize OIDC provider: %v", err)
	}

	oAuthMiddleware := middlewares.OAuthMiddleware()

	app.Use(cors.New(cors.Config{
		AllowHeaders:     "Origin, Content-Type, Accept, Content-Length, Accept-Language, Accept-Encoding, Connection, Access-Control-Allow-Origin, Authorization",
		AllowOrigins:     "*",
		AllowCredentials: false,
		AllowMethods:     "GET,POST,HEAD,PUT,DELETE,PATCH,OPTIONS,WebSocket,Upgrade",
	}))

	app.Static("/files", "files")

	routes.SetupRoutes(app, config, oAuthMiddleware)

	log.Fatal(app.Listen(":" + config.ServerConfig.Port))
}
