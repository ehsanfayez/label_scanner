package handlers

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"scanner/config"
	"scanner/internal/services"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type ReaderHandler struct {
	decodeService *services.DecryptService
	scanService   *services.ScanService
}

func NewReaderHandler(scanService *services.ScanService) *ReaderHandler {
	cfg := config.GetConfig()
	return &ReaderHandler{
		decodeService: services.NewDecryptService(cfg.ServerConfig.SecretKey),
		scanService:   scanService,
	}
}

type DecodeData struct {
	SerialNumber string `json:"serial_number"`
	InventoryID  string `json:"inventory_id"`
}

func validate(decodeService *services.DecryptService, token string) (*DecodeData, error) {
	// 16, 24, or 32 bytes for AES
	decrypted, err := decodeService.Decode(token)
	if err != nil {
		fmt.Printf("Error decoding data: %v\n", err)
		return nil, err
	}

	newData := &DecodeData{}
	if err := json.Unmarshal([]byte(decrypted), newData); err != nil {
		fmt.Printf("Error unmarshalling decrypted data: %v\n", err)
		return nil, err
	}

	fmt.Printf("%+v\n", newData)

	return newData, nil
}

func (h *ReaderHandler) Validate(c *fiber.Ctx) error {
	token := c.Params("token")
	_, err := validate(h.decodeService, token)
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Invalid token",
		})
	}

	return c.SendStatus(fiber.StatusOK)
}

func (h *ReaderHandler) Scan(c *fiber.Ctx) error {
	token := c.Params("token")
	data, err := validate(h.decodeService, token)
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
	ocrResponse, err := h.scanService.ScanFile(ImageType, files, "", data.InventoryID)
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
		"psid":  ocrResponse,
		"image": fileName,
	})
}

func (h *ReaderHandler) Store(c *fiber.Ctx) error {
	token := c.Params("token")
	data, err := validate(h.decodeService, token)
	if err != nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Invalid token",
		})
	}

	var requestData map[string]interface{}
	if err := c.BodyParser(&requestData); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to parse request body: %v", err),
		})
	}

	// check psid exit in requestData
	psid, ok := requestData["psid"].(string)
	if !ok {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "psid is required in the request body",
		})
	}

	image, _ := requestData["image"].(string)

	hardData := services.AddHardResponse{
		InventoryID:  data.InventoryID,
		SerialNumber: data.SerialNumber,
		Psid:         psid,
	}

	hard, err := h.scanService.AddHard(c.Context(), hardData, []string{image})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to store hard data: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"message":     "Hard data stored successfully",
		"hard_record": hard,
	})

}
