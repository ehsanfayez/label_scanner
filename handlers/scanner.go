package handlers

import (
	"bytes"
	"department/label_scanner/config"
	"department/label_scanner/services"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/gofiber/fiber/v2"
)

type ScanHandler struct {
	EmbeddingService *services.EmbeddingService
}

func NewScanHandler() *ScanHandler {
	return &ScanHandler{
		EmbeddingService: services.NewEmbeddingService(),
	}
}

type OCRResponse struct {
	Status    string                 `json:"status"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp" time_format:"2006-01-02 15:04:05.000000" time_location:"UTC"`
}

func (h *ScanHandler) Scan(c *fiber.Ctx) error {
	// get image from user
	file, err := c.FormFile("image")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Failed to get image",
		})
	}

	// open and read the file
	fileContent, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to open image",
		})
	}
	defer fileContent.Close()

	imageBytes, err := io.ReadAll(fileContent)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to read image",
		})
	}

	// convert image to base64
	base64Image := []byte(base64.StdEncoding.EncodeToString(imageBytes))
	// send image to ocr api
	ocrApi := config.GetConfig().OCRConfig.APIURL
	// add proxy to ocr api
	proxyUrl, err := url.Parse(config.GetConfig().ServerConfig.Proxy)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to parse proxy url",
		})
	}

	httpClient := &http.Client{
		Transport: &http.Transport{Proxy: http.ProxyURL(proxyUrl)},
		Timeout:   30 * time.Second,
	}

	data := map[string]interface{}{
		"image": base64Image,
	}

	dataBytes, err := json.Marshal(data)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to marshal data",
		})
	}

	resp, err := httpClient.Post(ocrApi, "application/json", bytes.NewBuffer(dataBytes))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to scan image",
		})
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to read response",
		})
	}

	var ocrResponse OCRResponse
	err = json.Unmarshal(body, &ocrResponse)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to unmarshal response",
		})
	}

	for key, value := range ocrResponse.Data {
		relatedWord := h.EmbeddingService.FindRelatedType(key)
		if relatedWord == "" {
			continue
		}

		ocrResponse.Data[relatedWord] = value
	}

	return c.JSON(ocrResponse)
}
