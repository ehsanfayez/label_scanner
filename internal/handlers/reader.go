package handlers

import (
	"context"
	"fmt"
	"path/filepath"
	"scanner/config"
	"scanner/internal/repositories"
	"scanner/internal/services"
	"slices"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type ReaderHandler struct {
	decodeService  *services.DecryptService
	scanService    *services.ScanService
	requestService *services.RequestService
}

func NewReaderHandler(scanService *services.ScanService, requestService *services.RequestService) *ReaderHandler {
	cfg := config.GetConfig()
	return &ReaderHandler{
		decodeService:  services.NewDecryptService(cfg.ServerConfig.SecretKey),
		scanService:    scanService,
		requestService: requestService,
	}
}

func validate(ctx context.Context, requestService *services.RequestService, token string) ([]string, error) {
	request, err := requestService.GetRequestByID(ctx, token)
	if err != nil || request == nil {
		return nil, fmt.Errorf("invalid token")
	}

	if len(request.SerialNumbers) == 0 {
		return nil, fmt.Errorf("invalid token")
	}

	return request.SerialNumbers, nil
}

func (h *ReaderHandler) Validate(c *fiber.Ctx) error {
	token := c.Params("token")
	serials, err := validate(c.Context(), h.requestService, token)
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Invalid token",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"serial_numbers": serials,
	})
}

func (h *ReaderHandler) Scan(c *fiber.Ctx) error {
	token := c.Params("token")
	serials, err := validate(c.Context(), h.requestService, token)
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Invalid token",
		})
	}

	contentType := c.Get("Content-Type")
	if contentType == "" || !strings.HasPrefix(strings.ToLower(contentType), "multipart/form-data") {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":        "Content-Type must be multipart/form-data",
			"received":     contentType,
			"content_type": c.Get("Content-Type"),
		})
	}

	//  get serial number from user request
	serialNumber := c.FormValue("serial_number")
	if !slices.Contains(serials, serialNumber) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Serial number not in the valid serial numbers list",
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

	files := form.File["image"]
	if len(files) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No images provided",
		})
	}

	ImageType := "hard"
	ocrResponse, err := h.scanService.ScanFile(ImageType, files, "", "")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	file := files[0]

	fileName := uuid.New().String() + filepath.Ext(file.Filename)
	// Save the file to disk (you can choose your own path)
	savePath := fmt.Sprintf("./uploads/%s", fileName)
	if err := c.SaveFile(file, savePath); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to save file: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"psid":  ocrResponse.Data["psid"],
		"image": fileName,
	})
}

type StoreRequest struct {
	Psid         string `json:"psid"`
	Image        string `json:"image"`
	SerialNumber string `json:"serial_number"`
}

func (h *ReaderHandler) Store(c *fiber.Ctx) error {
	token := c.Params("token")
	serials, err := validate(c.Context(), h.requestService, token)
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Invalid token",
		})
	}

	var requestData StoreRequest
	if err := c.BodyParser(&requestData); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to parse request body: %v", err),
		})
	}

	//  get serial number from user request
	if !slices.Contains(serials, requestData.SerialNumber) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Serial number not in the valid serial numbers list",
		})
	}

	hard, err := h.scanService.GetHardInfoByPsid(c.Context(), repositories.AddHardFilter{
		SerialNumber: requestData.SerialNumber,
		Psid:         requestData.Psid,
	})

	if err == nil && hard != nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "Hard with the same PSID and Serial Number already exists",
		})
	}

	// check psid exit in requestData
	psid := requestData.Psid
	image := requestData.Image

	hardData := services.AddHardResponse{
		SerialNumber: requestData.SerialNumber,
		Psid:         psid,
	}

	_, err = h.scanService.AddHard(c.Context(), hardData, []string{image})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to store hard data: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Hard data stored successfully",
	})
}
