package handlers

import (
	"scanner/internal/services"

	"github.com/gofiber/fiber/v2"
)

type ScanHandler struct {
	ScanService *services.ScanService
}

func NewScanHandler() *ScanHandler {
	return &ScanHandler{ScanService: services.NewScanService()}
}

func (h *ScanHandler) Scan(c *fiber.Ctx) error {
	// get image from user
	form, err := c.MultipartForm()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Failed to get image",
		})
	}

	files := form.File["image"]
	if len(files) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No images provided",
		})
	}

	ImageType := c.FormValue("type")
	if ImageType == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Image type is required",
		})
	}

	ocrResponse, err := h.ScanService.Scan(ImageType, files, "")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(ocrResponse)
}

func (h *ScanHandler) ScanType(c *fiber.Ctx) error {
	// get image from user
	form, err := c.MultipartForm()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Failed to get image",
		})
	}

	files := form.File["image"]
	if len(files) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No images provided",
		})
	}

	// get image type from user
	imageType := c.FormValue("type")
	if imageType == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Image type is required",
		})
	}

	Sender := c.FormValue("sender")

	ocrResponse, err := h.ScanService.Scan(imageType, files, Sender)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(ocrResponse)
}
