package services

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"scanner/config"
	"strings"
	"time"
)

type ScanService struct {
}

func NewScanService() *ScanService {
	return &ScanService{}
}

type OCRResponse struct {
	Status    string            `json:"status"`
	Data      map[string]string `json:"data"`
	Timestamp string            `json:"timestamp"`
}

func (s *ScanService) Scan(ImageType string, files []*multipart.FileHeader, Sender string, InventoryId string) (*OCRResponse, error) {
	base64Images := []string{}
	for _, file := range files {
		// open and read the file
		fileContent, err := file.Open()
		if err != nil {
			return nil, errors.New("failed to open image")

		}

		defer fileContent.Close()

		imageBytes, err := io.ReadAll(fileContent)
		if err != nil {
			return nil, errors.New("failed to read image")
		}

		// convert image to base64
		base64Image := base64.StdEncoding.EncodeToString(imageBytes)

		// send image to ocr api
		// should remove data:image/jpeg;base64,
		base64Image = strings.ReplaceAll(string(base64Image), "data:image/jpeg;base64,", "")
		base64Images = append(base64Images, base64Image)
	}

	ocrApi := config.GetConfig().OCRConfig.APIURL + "/scan"
	// add proxy to ocr api
	proxyUrl, err := url.Parse(config.GetConfig().ServerConfig.Proxy)
	if err != nil {
		return nil, errors.New("failed to parse proxy url")
	}

	httpClient := &http.Client{
		Transport: &http.Transport{Proxy: http.ProxyURL(proxyUrl)},
		Timeout:   120 * time.Second,
	}

	data := map[string]interface{}{
		"images": base64Images,
	}

	if ImageType != "" {
		data["type"] = ImageType
	}

	data["inventory_id"] = InventoryId
	dataBytes, err := json.Marshal(data)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("failed to marshal data")
	}

	req, err := http.NewRequest("POST", ocrApi, bytes.NewBuffer(dataBytes))
	if err != nil {
		return nil, errors.New("failed to create request")
	}

	req.Header.Set(config.GetConfig().OCRConfig.APIHeader, config.GetConfig().OCRConfig.APIKey)
	resp, err := httpClient.Do(req)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("failed to scan image")
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.New("failed to read response")
	}

	fmt.Println("body", string(body))

	var ocrResponse OCRResponse
	err = json.Unmarshal(body, &ocrResponse)
	if err != nil {
		return nil, errors.New("failed to unmarshal response")
	}

	if Sender == "scanner" {
		for key, value := range ocrResponse.Data {
			if dat, ok := ocrResponse.Data["capacity"]; ok {
				dat = strings.ToUpper(dat)
				if strings.Contains(dat, "GB") {
					ocrResponse.Data["capacity"] = strings.TrimSpace(strings.ReplaceAll(dat, "GB", ""))
					ocrResponse.Data["unit"] = "GB"
				}

				if strings.Contains(dat, "TB") {
					ocrResponse.Data["capacity"] = strings.TrimSpace(strings.ReplaceAll(dat, "TB", ""))
					ocrResponse.Data["unit"] = "TB"
				}
			}

			if dat, ok := ocrResponse.Data["hard_type"]; ok {
				ocrResponse.Data["type"] = strings.ToUpper(dat)
				ocrResponse.Data["hard_type"] = strings.ToUpper(dat)
			}

			if dat, ok := ocrResponse.Data["ram_type"]; ok {
				ocrResponse.Data["type"] = strings.ToUpper(dat)
				ocrResponse.Data["ram_type"] = strings.ToUpper(dat)
			}

			if dat, ok := ocrResponse.Data["type"]; ok {
				ocrResponse.Data["type"] = strings.ToUpper(dat)
			}

			if key == "brand" {
				ocrResponse.Data["make"] = value
				continue
			}
		}
	} else {
		for key, value := range ocrResponse.Data {
			if key == "brand" {
				ocrResponse.Data["make"] = value
				continue
			}
		}
	}

	return &ocrResponse, nil
}
