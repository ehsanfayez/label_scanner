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
	ScanService    *services.ScanService
	RequestService *services.RequestService
}

func NewWebServiceHandler(scanService *services.ScanService, requestService *services.RequestService) *WebServiceHandler {
	return &WebServiceHandler{
		ScanService:    scanService,
		RequestService: requestService,
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
		hard.Images[index] = cfg.ServerConfig.BaseUrl + "/image/" + image
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
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to parse query parameters: %v", err),
		})
	}

	hards, err := h.ScanService.GetHardInfoByHardFilter(c.Context(), &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to get hard info: %v", err),
		})
	}

	cfg := config.GetConfig()
	for idx, hard := range hards {
		images := []string{}
		for _, image := range hard.Images {
			images = append(images, cfg.ServerConfig.BaseUrl+"/image/"+image)
		}

		hards[idx].Images = images
	}

	return c.JSON(fiber.Map{
		"status":    "success",
		"data":      hards,
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

	if req.Psid == "" || req.SerialNumber == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Psid and SerialNumber are required",
		})
	}

	//check if exit error
	hard, err := h.ScanService.GetHardInfoByPsid(c.Context(), repositories.AddHardFilter{
		Psid:         req.Psid,
		SerialNumber: req.SerialNumber,
	})

	if err == nil && hard != nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": "Hard with the same InventoryID, Make, and SerialNumber already exists",
		})
	}

	hard, err = h.ScanService.AddHard(c.Context(), req, []string{})
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
	hardID := c.Params("id")
	var req services.EditHardResponse
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to parse request body: %v", err),
		})
	}

	hard, err := h.ScanService.GetHardInfo(c.Context(), hardID)
	if err != nil || hard == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Hard not found",
		})
	}

	// update
	err = h.ScanService.UpdateHard(c.Context(), hard, req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to update hard: %v", err),
		})
	}

	if len(hard.Images) > 0 {
		cfg := config.GetConfig()
		for index, image := range hard.Images {
			hard.Images[index] = cfg.ServerConfig.BaseUrl + "/image/" + image
		}
	}

	return c.JSON(fiber.Map{
		"status":    "success",
		"data":      hard,
		"timestamp": time.Now(),
	})
}

type VipeAcceptRequest struct {
	SerialNumber string `json:"serial_number" form:"serial_number"`
	Psid         string `json:"psid" form:"psid"`
}

func (h *WebServiceHandler) VipeAccept(c *fiber.Ctx) error {
	var req VipeAcceptRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to parse request body: %v", err),
		})
	}

	err := h.ScanService.VipeAccept(c.Context(), req.SerialNumber, req.Psid)
	if err != nil {
		fmt.Println(err.Error())
		if strings.Contains(err.Error(), "no documents in result") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Hard not found",
			})
		}

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to process vipe accept",
		})
	}

	return c.JSON(fiber.Map{
		"status":    "success",
		"message":   "Vipe accept processed successfully",
		"timestamp": time.Now(),
	})
}

type GeneratePsidUrlRequest struct {
	SerialNumbers []string `json:"serial_numbers" form:"serial_numbers"`
}

func (h *WebServiceHandler) GeneratePsidUrl(c *fiber.Ctx) error {
	var req GeneratePsidUrlRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to parse request body: %v", err),
		})
	}

	request, err := h.RequestService.FindExistingSerialNumbers(c.Context(), req.SerialNumbers)
	if err != nil && err.Error() != "mongo: no documents in result" {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to check existing serial numbers",
		})
	}

	if request != nil {
		return c.JSON(fiber.Map{
			"status":     "success",
			"request_id": request.UUid,
			"timestamp":  time.Now(),
		})
	}

	psidUrl, err := h.RequestService.CreateRequest(c.Context(), req.SerialNumbers)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to generate PSID URLs: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"status":     "success",
		"request_id": psidUrl.UUid,
		"timestamp":  time.Now(),
	})
}

type DeletePsidRequest struct {
	SerialNumber string `json:"serial_number" form:"serial_number"`
	Psid         string `json:"psid" form:"psid"`
}

func (h *WebServiceHandler) DeletePsid(c *fiber.Ctx) error {
	var req DeletePsidRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to parse request body: %v", err),
		})
	}

	hard, err := h.ScanService.GetHardInfoByPsid(c.Context(), repositories.AddHardFilter{
		SerialNumber: req.SerialNumber,
		Psid:         req.Psid,
	})

	if err != nil || hard == nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Hard not found",
		})
	}

	// update
	err = h.ScanService.DeletePsid(c.Context(), hard)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": fmt.Sprintf("Failed to delete psid: %v", err),
		})
	}

	return c.JSON(fiber.Map{
		"status":    "success",
		"message":   "Psid deleted successfully",
		"timestamp": time.Now(),
	})

}
