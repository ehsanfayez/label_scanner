package main

import (
	"context"
	"log"
	"scanner/config"
	"scanner/internal/handlers"
	"scanner/internal/middlewares"
	"scanner/internal/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

var types = map[string]map[string]string{
	"laptop": {
		"make":               "Make",
		"model":              "Model",
		"cpu_model":          "CPU Model",
		"cpu_series":         "CPU Series",
		"serial_number":      "Serial Number",
		"part_number":        "Part Number",
		"battery":            "Battery",
		"adapter":            "Adapter",
		"rams":               "RAM Full",
		"screen_size_inches": "Screen Size Inches",
		"hdd_capacity_size":  "HDD Capacity Size",
		"ram_capacity_size":  "Ram Capacity Size",
		"hdd_type":           "HDD Type",
		"ram_type":           "Ram Type",
		"hdds":               "HDD Full",
		"cpu_speed":          "CPU Speed",
		"gpu_model":          "GPU Model",
		"cam":                "Cam",
	},
	"desktop": {
		"make":               "Make",
		"model":              "Model",
		"cpu_model":          "CPU Model",
		"cpu_series":         "CPU Series",
		"serial_number":      "Serial Number",
		"part_number":        "Part Number",
		"battery":            "Battery",
		"adapter":            "Adapter",
		"rams":               "RAM Full",
		"screen_size_inches": "Screen Size Inches",
		"hdds":               "HDD Full",
		"hdd_capacity_size":  "HDD Capacity Size",
		"ram_capacity_size":  "Ram Capacity Size",
		"hdd_type":           "HDD Type",
		"ram_type":           "Ram Type",
		"cpu_speed":          "CPU Speed",
		"gpu_model":          "GPU Model",
		"cam":                "Cam",
	},
	"server": {
		"make":               "Make",
		"model":              "Model",
		"cpu_model":          "CPU Model",
		"cpu_series":         "CPU Series",
		"serial_number":      "Serial Number",
		"part_number":        "Part Number",
		"battery":            "Battery",
		"adapter":            "Adapter",
		"rams":               "RAM Full",
		"hdd_capacity_size":  "HDD Capacity Size",
		"ram_capacity_size":  "Ram Capacity Size",
		"hdd_type":           "HDD Type",
		"ram_type":           "Ram Type",
		"screen_size_inches": "Screen Size Inches",
		"hdds":               "HDD Full",
		"cpu_speed":          "CPU Speed",
		"gpu_model":          "GPU Model",
		"cam":                "Cam",
	},
}

var rams = []string{
	"DDR3",
	"DDR4",
	"DDR5",
	"LPDDR3",
	"LPDDR4",
	"LPDDR5",
}

var hards = []string{
	"SATA 2.5",
	"SSD 2.5",
	"NVME SSD",
}

func main() {
	config := config.InitConfig()
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

	app.Get("/api/types", oAuthMiddleware, func(c *fiber.Ctx) error {
		return c.JSON(types)
	})

	app.Get("/api/storages", oAuthMiddleware, func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"rams":  rams,
			"hards": hards,
		})
	})

	app.Get("/api/types/:type", oAuthMiddleware, func(c *fiber.Ctx) error {
		return c.JSON(types[c.Params("type")])
	})

	scanHandler := handlers.NewScanHandler()
	app.Post("/api/scan", oAuthMiddleware, scanHandler.Scan)

	app.Post("/api/scan_type", oAuthMiddleware, scanHandler.ScanType)

	dataHandler := handlers.NewDataHandler()
	app.Post("/api/done", oAuthMiddleware, dataHandler.Done)

	log.Fatal(app.Listen(":" + config.ServerConfig.Port))
}
