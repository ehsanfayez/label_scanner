package services

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"scanner/config"
	"scanner/internal/repositories"
	"slices"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ScanService struct {
	hardRepo *repositories.HardRepository
}

func NewScanService() *ScanService {
	return &ScanService{
		hardRepo: repositories.NewHardRepository(),
	}
}

type OCRResponse struct {
	Status    string                 `json:"status"`
	Data      map[string]interface{} `json:"data"`
	Timestamp string                 `json:"timestamp"`
	ImageUrl  []string               `json:"image_url"`
}

func (s *ScanService) ScanFile(ImageType string, files []*multipart.FileHeader, Sender string, InventoryId string) (*OCRResponse, error) {
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

	return s.Scan(ImageType, base64Images, Sender, InventoryId)
}

func (s *ScanService) Scan(ImageType string, base64Images []string, Sender string, InventoryId string) (*OCRResponse, error) {
	cfg := config.GetConfig()
	ocrApi := cfg.OCRConfig.APIURL + "/scan"
	fmt.Println(ocrApi)
	// add proxy to ocr api

	// Force IPv4 by using custom dialer
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return dialer.DialContext(ctx, "tcp4", addr)
			},
		},
		Timeout: 120 * time.Second,
	}

	if cfg.ServerConfig.ProxyScan {
		proxyUrl, err := url.Parse(cfg.ServerConfig.Proxy)
		if err != nil {
			return nil, errors.New("failed to parse proxy url")
		}

		httpClient.Transport = &http.Transport{
			Proxy: http.ProxyURL(proxyUrl),
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return dialer.DialContext(ctx, "tcp4", addr)
			},
		}
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
		fmt.Println(err)
		return nil, errors.New("failed to create request")
	}

	req.Header.Set(cfg.OCRConfig.APIHeader, cfg.OCRConfig.APIKey)
	fmt.Println(cfg.OCRConfig.APIHeader, cfg.OCRConfig.APIKey)
	resp, err := httpClient.Do(req)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("failed to scan image")
	}

	fmt.Println(resp.StatusCode)

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("failed to read response")
	}

	var ocrResponse OCRResponse
	err = json.Unmarshal(body, &ocrResponse)
	if err != nil {
		fmt.Println(err)
		return nil, errors.New("failed to unmarshal response")
	}

	fmt.Printf("ocr response: %+v\n", ocrResponse)

	if Sender == "scanner" {
		for key, value := range ocrResponse.Data {
			if dat, ok := ocrResponse.Data["capacity"]; ok {
				if datStr, ok := dat.(string); ok {
					datStr = strings.ToUpper(datStr)
					if strings.Contains(datStr, "GB") {
						ocrResponse.Data["capacity"] = strings.TrimSpace(strings.ReplaceAll(datStr, "GB", ""))
						ocrResponse.Data["unit"] = "GB"
					}

					if strings.Contains(datStr, "TB") {
						ocrResponse.Data["capacity"] = strings.TrimSpace(strings.ReplaceAll(datStr, "TB", ""))
						ocrResponse.Data["unit"] = "TB"
					}
				}
			}

			if dat, ok := ocrResponse.Data["hard_type"]; ok {
				if datStr, ok := dat.(string); ok {
					ocrResponse.Data["type"] = strings.ToUpper(datStr)
					ocrResponse.Data["hard_type"] = strings.ToUpper(datStr)
				}
			}

			if dat, ok := ocrResponse.Data["ram_type"]; ok {
				if datStr, ok := dat.(string); ok {
					ocrResponse.Data["type"] = strings.ToUpper(datStr)
					ocrResponse.Data["ram_type"] = strings.ToUpper(datStr)
				}
			}

			if dat, ok := ocrResponse.Data["type"]; ok {
				if datStr, ok := dat.(string); ok {
					ocrResponse.Data["type"] = strings.ToUpper(datStr)
				}
			}

			if key == "brand" {
				if datStr, ok := value.(string); ok {
					ocrResponse.Data["make"] = strings.ToUpper(datStr)
				}
			}
		}
	} else {
		for key, value := range ocrResponse.Data {
			if key == "brand" {
				if datStr, ok := value.(string); ok {
					ocrResponse.Data["make"] = strings.ToUpper(datStr)
				}
			}
		}
	}

	return &ocrResponse, nil
}

func (s *ScanService) StoreScanResultIfNotExists(ctx context.Context, ocrResponse *OCRResponse, images []string, inventoryId string) (*repositories.Hard, error) {
	serialNumber := ""
	psidValue := ""

	if val, ok := ocrResponse.Data["serial_number"]; ok {
		if valStr, ok := val.(string); ok {
			serialNumber = valStr
		}
	}

	if val, ok := ocrResponse.Data["psid"]; ok {
		if valStr, ok := val.(string); ok {
			psidValue = valStr
		}
	}

	existingHard, err := s.hardRepo.FindByPsid(ctx, repositories.AddHardFilter{
		SerialNumber: serialNumber,
		Psid:         psidValue,
	})

	if err != nil && err.Error() != "mongo: no documents in result" {
		return nil, err
	}

	if existingHard != nil {
		// already exists
		return existingHard, nil
	}

	newHard := &repositories.Hard{
		ID:           primitive.NewObjectID(),
		Capacity:     "",
		Eui:          "",
		Type:         "",
		InventoryID:  inventoryId,
		Make:         "",
		Model:        "",
		PartNumber:   "",
		SerialNumber: serialNumber,
		Psid:         psidValue,
		ExtraFileds:  make(map[string]interface{}),
		Images:       images,
	}

	for key, value := range ocrResponse.Data {
		valStr := ""
		if vl, ok := value.(string); ok {
			valStr = fmt.Sprintf("%v", vl)
		}

		switch key {
		case "capacity":
			newHard.Capacity = valStr
		case "eui":
			newHard.Eui = valStr
		case "hard_type":
			newHard.Type = valStr
		case "model":
			newHard.Model = valStr
		case "make":
			newHard.Make = valStr
		case "part_number":
			newHard.PartNumber = valStr
		case "psid":
			newHard.Psid = valStr
		default:
			if slices.Contains([]string{"make", "serial_number", "inventory_id", "hard_id"}, key) {
				continue
			}

			newHard.ExtraFileds[key] = value
		}
	}

	err = s.hardRepo.Insert(ctx, newHard)
	if err != nil {
		return nil, err
	}

	return newHard, nil
}

func (s *ScanService) GetHardInfoByHardFilter(ctx context.Context, filter *repositories.HardFilter) ([]repositories.Hard, error) {
	hards, err := s.hardRepo.FindByInput(ctx, filter)
	if err != nil {
		return nil, err
	}

	return hards, nil
}

func (s *ScanService) GetHardInfo(ctx context.Context, id string) (*repositories.Hard, error) {
	hard, err := s.hardRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return hard, nil
}

func (s *ScanService) DeletePsid(ctx context.Context, hard *repositories.Hard) error {
	return s.hardRepo.DeleteByPsid(ctx, hard)
}

func (s *ScanService) GetHardInfoByPsid(ctx context.Context, filter repositories.AddHardFilter) (*repositories.Hard, error) {
	hard, err := s.hardRepo.FindByPsid(ctx, filter)
	if err != nil {
		return nil, err
	}

	return hard, nil
}

type AddHardResponse struct {
	InventoryID  string `json:"inventory_id" form:"inventory_id"`
	Type         string `json:"hard_type" form:"hard_type"`
	Capacity     string `json:"capacity" form:"capacity"`
	Eui          string `json:"eui" form:"eui"`
	Make         string `json:"make" form:"make"`
	Model        string `json:"model" form:"model"`
	PartNumber   string `json:"part_number" form:"part_number"`
	SerialNumber string `json:"serial_number" form:"serial_number"`
	Psid         string `json:"psid" form:"psid"`
}

func (s *ScanService) AddHard(ctx context.Context, data AddHardResponse, images []string) (*repositories.Hard, error) {
	newHard := &repositories.Hard{
		ID:           primitive.NewObjectID(),
		Capacity:     data.Capacity,
		Eui:          data.Eui,
		Type:         data.Type,
		InventoryID:  data.InventoryID,
		Make:         data.Make,
		Model:        data.Model,
		PartNumber:   data.PartNumber,
		SerialNumber: data.SerialNumber,
		Psid:         data.Psid,
		ExtraFileds:  make(map[string]interface{}),
		Images:       images,
	}

	err := s.hardRepo.Insert(ctx, newHard)
	if err != nil {
		return nil, err
	}

	return newHard, nil
}

type EditHardResponse struct {
	InventoryID  *string `json:"inventory_id" form:"inventory_id"`
	Type         *string `json:"hard_type" form:"hard_type"`
	Capacity     *string `json:"capacity" form:"capacity"`
	Eui          *string `json:"eui" form:"eui"`
	Make         *string `json:"make" form:"make"`
	Model        *string `json:"model" form:"model"`
	PartNumber   *string `json:"part_number" form:"part_number"`
	SerialNumber *string `json:"serial_number" form:"serial_number"`
	Psid         *string `json:"psid" form:"psid"`
}

func (s *ScanService) UpdateHard(ctx context.Context, hard *repositories.Hard, data EditHardResponse) error {
	if data.Capacity != nil {
		hard.Capacity = *data.Capacity
	}

	if data.Eui != nil {
		hard.Eui = *data.Eui
	}

	if data.Type != nil {
		hard.Type = *data.Type
	}

	if data.InventoryID != nil {
		hard.InventoryID = *data.InventoryID
	}

	if data.Make != nil {
		hard.Make = *data.Make
	}

	if data.Model != nil {
		hard.Model = *data.Model
	}

	if data.PartNumber != nil {
		hard.PartNumber = *data.PartNumber
	}

	if data.SerialNumber != nil {
		hard.SerialNumber = *data.SerialNumber
	}

	if data.Psid != nil {
		hard.Psid = *data.Psid
	}

	hard.UserEdited = true

	return s.hardRepo.Update(ctx, hard.ID.Hex(), hard)
}

func (s *ScanService) WipeAccept(ctx context.Context, serialNumber, psid string) error {
	hard, err := s.hardRepo.FindByPsid(ctx, repositories.AddHardFilter{
		SerialNumber: serialNumber,
		Psid:         psid,
	})

	if err != nil {
		if err.Error() == "mongo: no documents in result" {
			newHard := &repositories.Hard{
				ID:           primitive.NewObjectID(),
				SerialNumber: serialNumber,
				Psid:         psid,
				ExtraFileds:  make(map[string]interface{}),
				WipeAccepted: true,
			}

			err := s.hardRepo.Insert(ctx, newHard)
			if err != nil {
				return err
			}

			return nil
		}

		return err
	}

	hard.WipeAccepted = true
	return s.hardRepo.WipeAccepted(ctx, hard)
}
