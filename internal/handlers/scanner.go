package handlers

import (
	"fmt"
	"scanner/internal/services"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type ScanHandler struct {
	ScanService *services.ScanService
}

func NewScanHandler() *ScanHandler {
	return &ScanHandler{ScanService: services.NewScanService()}
}

func (h *ScanHandler) Scan(c *fiber.Ctx) error {
	// Check Content-Type header
	contentType := c.Get("Content-Type")
	fmt.Printf("Received Content-Type: '%s'\n", contentType)
	fmt.Printf("All headers: %v\n", c.GetReqHeaders())
	
	if contentType == "" || !strings.HasPrefix(strings.ToLower(contentType), "multipart/form-data") {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":        "Content-Type must be multipart/form-data",
			"received":     contentType,
			"content_type": c.Get("Content-Type"),
		})
	}

	// get image from user
	form, err := c.MultipartForm()
	if err != nil {
		fmt.Println("Multipart form error:", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to parse multipart form: %v", err),
		})
	}

	files := form.File["images"]
	if len(files) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No images provided",
		})
	}

	ImageType := "hard"

	ocrResponse, err := h.ScanService.Scan(ImageType, files, "")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(ocrResponse)
}

func (h *ScanHandler) ScanType(c *fiber.Ctx) error {
	// Check Content-Type header
	contentType := c.Get("Content-Type")
	fmt.Printf("Received Content-Type: '%s'\n", contentType)
	fmt.Printf("All headers: %v\n", c.GetReqHeaders())
	
	if contentType == "" || !strings.HasPrefix(strings.ToLower(contentType), "multipart/form-data") {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":        "Content-Type must be multipart/form-data",
			"received":     contentType,
			"content_type": c.Get("Content-Type"),
		})
	}

	// get image from user
	form, err := c.MultipartForm()
	if err != nil {
		fmt.Println("Multipart form error:", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to parse multipart form: %v", err),
		})
	}

	files := form.File["images"]
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
