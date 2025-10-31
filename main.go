package main

import (
	"log"
	"scanner/config"
	"scanner/handlers"

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
		"screen_size_inches": "Screen Size Inches",
		"hdds":               "HDD Full",
		"cpu_speed":          "CPU Speed",
		"gpu_model":          "GPU Model",
		"cam":                "Cam",
	},
}

func main() {
	config := config.InitConfig()
	app := fiber.New()

	app.Use(cors.New(cors.Config{
		AllowHeaders:     "Origin, Content-Type, Accept, Content-Length, Accept-Language, Accept-Encoding, Connection, Access-Control-Allow-Origin, Authorization",
		AllowOrigins:     "*",
		AllowCredentials: false,
		AllowMethods:     "GET,POST,HEAD,PUT,DELETE,PATCH,OPTIONS,WebSocket,Upgrade",
	}))

	app.Static("/files", "files")
	app.Post("/api/login", handlers.Login)

	app.Get("/api/types", func(c *fiber.Ctx) error {
		return c.JSON(types)
	})

	app.Get("/api/types/:type", func(c *fiber.Ctx) error {
		return c.JSON(types[c.Params("type")])
	})

	scanHandler := handlers.NewScanHandler()
	app.Post("/api/scan", scanHandler.Scan)

	dataHandler := handlers.NewDataHandler()
	app.Post("/api/done", dataHandler.Done)

	log.Fatal(app.Listen(":" + config.ServerConfig.Port))
}
