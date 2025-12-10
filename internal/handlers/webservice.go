package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"scanner/config"
	"scanner/internal/repositories"
	"scanner/internal/services"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type WebServiceHandler struct {
	ScanService *services.ScanService
}

func NewWebServiceHandler() *WebServiceHandler {
	return &WebServiceHandler{
		ScanService: services.NewScanService(),
	}
}

func (h *WebServiceHandler) HealthCheck(c *fiber.Ctx) error {
	cfg := config.GetConfig()
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("GET", cfg.OCRConfig.APIURL, nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status": "unhealthy",
			"error":  "failed to build request",
		})
	}

	resp, err := client.Do(req)
	if err != nil {
		// handle timeout explicitly
		var netErr net.Error
		if errors.Is(err, context.DeadlineExceeded) ||
			(errors.As(err, &netErr) && netErr.Timeout()) {

			return c.Status(fiber.StatusGatewayTimeout).JSON(fiber.Map{
				"status": "unhealthy",
				"error":  "OCR service timeout",
			})
		}

		// all other request errors
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status": "unhealthy",
			"error":  "OCR service unreachable",
		})
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status": "unhealthy",
			"error":  "failed to read OCR response",
		})
	}

	data := map[string]interface{}{}
	if err := json.Unmarshal(body, &data); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status": "unhealthy",
			"error":  "failed to parse OCR service response",
		})
	}

	// expected behavior: service returns {"detail": "Not Found"} on GET /
	if data["detail"] != "Not Found" {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status": "unhealthy",
			"error":  "OCR service unexpected response",
		})
	}

	return c.SendStatus(fiber.StatusOK)
}

type ScanRequest struct {
	Images      []string `json:"images" form:"images"`
	InventoryID string   `json:"inventory_id" form:"inventory_id"`
	Type        string   `json:"type" form:"type"`
}

func (h *WebServiceHandler) Scan(c *fiber.Ctx) error {
	var scanReq ScanRequest
	if err := c.BodyParser(&scanReq); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to parse request body: %v", err),
		})
	}

	if len(scanReq.Images) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No images provided",
		})
	}

	ImageType := scanReq.Type
	if ImageType == "" {
		ImageType = "hard"
	}

	ocrResponse, err := h.ScanService.Scan(ImageType, scanReq.Images, "", scanReq.InventoryID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	//convert base64 images to image files and store paths in mongo db
	imagePaths := []string{}
	for idx, base64Image := range scanReq.Images {
		imageData, err := base64.StdEncoding.DecodeString(base64Image)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fmt.Sprintf("invalid base64 image at index %d: %v", idx, err),
			})
		}

		fileName := uuid.New().String() + ".jpg"
		savePath := fmt.Sprintf("./uploads/%s", fileName)

		if err := os.WriteFile(savePath, imageData, 0644); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to save file: %v", err),
			})
		}

		imagePaths = append(imagePaths, fileName)
		fmt.Println("Saved image", idx, "to", savePath)
	}

	// store response in mongo db
	hard, err := h.ScanService.StoreScanResultIfNotExists(c.Context(), ocrResponse, imagePaths, scanReq.InventoryID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to store scan result: %v", err),
		})
	}

	cfg := config.GetConfig()
	for index, image := range hard.Images {
		hard.Images[index] = cfg.ServerConfig.BaseUrl + "/images/" + image
	}

	return c.JSON(fiber.Map{
		"staus":     "success",
		"data":      hard,
		"timestamp": time.Now(),
	})
}

type ScanFileRequest struct {
	Images      []*multipart.FileHeader `form:"images" json:"images"`
	InventoryID string                  `form:"inventory_id" json:"inventory_id"`
	Type        string                  `form:"type,omitempty" json:"type,omitempty"`
}

func (h *WebServiceHandler) ScanFile(c *fiber.Ctx) error {
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

	files := form.File["images"]
	if len(files) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "No images provided",
		})
	}

	ImageType := "hard"
	inventoryId := c.FormValue("inventory_id")
	ocrResponse, err := h.ScanService.ScanFile(ImageType, files, "", inventoryId)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// store images and store paths in mongo db
	images := []string{}
	for _, file := range files {
		fileName := uuid.New().String() + filepath.Ext(file.Filename)
		// Save the file to disk (you can choose your own path)
		savePath := fmt.Sprintf("./uploads/%s", fileName)
		if err := c.SaveFile(file, savePath); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to save file: %v", err),
			})
		}

		images = append(images, fileName)
	}

	// store response in mongo db
	hard, err := h.ScanService.StoreScanResultIfNotExists(c.Context(), ocrResponse, images, inventoryId)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to store scan result: %v", err),
		})
	}

	cfg := config.GetConfig()
	for index, image := range hard.Images {
		hard.Images[index] = cfg.ServerConfig.BaseUrl + "/images/" + image
	}

	return c.JSON(fiber.Map{
		"staus":     "success",
		"data":      hard,
		"timestamp": time.Now(),
	})
}

func (h *WebServiceHandler) GetImage(c *fiber.Ctx) error {
	filename := c.Params("filename")
	filePath := fmt.Sprintf("./uploads/%s", filename)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Image not found",
		})
	}

	// Serve the file
	return c.SendFile(filePath)
}

func (h *WebServiceHandler) GetInfo(c *fiber.Ctx) error {
	var req repositories.HardFilter
	if err := c.QueryParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to parse query parameters: %v", err),
		})
	}

	hard, err := h.ScanService.GetHardInfo(c.Context(), req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to get hard info: %v", err),
		})
	}

	cfg := config.GetConfig()
	for index, image := range hard.Images {
		hard.Images[index] = cfg.ServerConfig.BaseUrl + "/images/" + image
	}

	return c.JSON(fiber.Map{
		"status":    "success",
		"data":      hard,
		"timestamp": time.Now(),
	})
}

func (h *WebServiceHandler) ScanType(c *fiber.Ctx) error {
	return nil
}

func (h *WebServiceHandler) AddHard(c *fiber.Ctx) error {
	var req services.AddHardResponse
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to parse request body: %v", err),
		})
	}

	//check if exit error
	hard, err := h.ScanService.GetHardInfo(c.Context(), repositories.HardFilter{
		InventoryID:  req.InventoryID,
		Make:         req.Make,
		SerialNumber: req.SerialNumber,
	})

	if err == nil && hard != nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "Hard with the same InventoryID, Make, and SerialNumber already exists",
		})
	}

	hard, err = h.ScanService.AddHard(c.Context(), req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to add hard: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"status":    "success",
		"data":      hard,
		"timestamp": time.Now(),
	})
}

func (h *WebServiceHandler) EditHard(c *fiber.Ctx) error {
	var req services.EditHardResponse
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to parse request body: %v", err),
		})
	}

	inventiryID := ""
	if req.InventoryID != nil {
		inventiryID = *req.InventoryID
	}

	make := ""
	if req.Make != nil {
		make = *req.Make
	}

	serialNumber := ""
	if req.SerialNumber != nil {
		serialNumber = *req.SerialNumber
	}

	hard, err := h.ScanService.GetHardInfo(c.Context(), repositories.HardFilter{
		InventoryID:  inventiryID,
		Make:         make,
		SerialNumber: serialNumber,
	})

	if err != nil || hard == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to get hard info: %v", err),
		})
	}

	err = h.ScanService.UpdateHard(c.Context(), hard, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to update hard info: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"status":    "success",
		"data":      hard,
		"timestamp": time.Now(),
	})
}
