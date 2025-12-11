package routes

import (
	"scanner/config"
	"scanner/internal/handlers"
	"scanner/internal/middlewares"
	"scanner/internal/services"

	"github.com/gofiber/fiber/v2"
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

func SetupRoutes(app *fiber.App, config *config.Config, oAuthMiddleware fiber.Handler) {
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
	app.Post("/api/scan", scanHandler.Scan)

	app.Post("/api/scan_type", oAuthMiddleware, scanHandler.ScanType)

	dataHandler := handlers.NewDataHandler()
	app.Post("/api/done", oAuthMiddleware, dataHandler.Done)
	scanService := services.NewScanService()
	SetupWebServicesRoutes(app, config, scanService)
	SetupReaderRoutes(app, scanService)
}

func SetupWebServicesRoutes(app *fiber.App, config *config.Config, scanService *services.ScanService) {
	webServiceHandler := handlers.NewWebServiceHandler(scanService)
	webserviceMiddleware := middlewares.WebserviceMiddleware()
	app.Get("/api/webservice/health", webserviceMiddleware, webServiceHandler.HealthCheck)
	app.Post("/api/webservice/scan", webserviceMiddleware, webServiceHandler.Scan)
	app.Post("/api/webservice/scan_file", webserviceMiddleware, webServiceHandler.ScanFile)
	app.Get("/api/webservice/hards", webserviceMiddleware, webServiceHandler.GetInfo)
	app.Get("/image/:filename", webServiceHandler.GetImage)
	app.Post("/api/webservice/hards", webserviceMiddleware, webServiceHandler.AddHard)
	app.Put("/api/webservice/hards", webserviceMiddleware, webServiceHandler.EditHard)
}

func SetupReaderRoutes(app *fiber.App, scanService *services.ScanService) {
	readerHandler := handlers.NewReaderHandler(scanService)
	app.Post("/api/reader/validate/:token", readerHandler.Validate)
	app.Post("/api/reader/scan/:token", readerHandler.Scan)
	app.Post("/api/reader/store/:token", readerHandler.Store)
}
