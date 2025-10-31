package handlers

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
)

type DataHandler struct {
}

func NewDataHandler() *DataHandler {
	return &DataHandler{}
}

var FieldKeys = []string{
	"job_number",
	"type",
	"make",
	"model",
	"cpu_model",
	"cpu_series",
	"serial_number",
	"part_number",
	"battery",
	"adapter",
	"rams",
	"screen_size_inches",
	"hdds",
	"cpu_speed",
	"gpu_model",
	"cam",
}

var Fields = map[string]string{
	"job_number":         "Job Number",
	"type":               "Type",
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
	"cpu_speed":          "CPU Speed",
	"gpu_model":          "GPU Model",
	"cam":                "Cam",
}

type Data struct {
	JobNumber        string    `json:"job_number"`
	Type             string    `json:"type"`
	Make             string    `json:"make"`
	Model            string    `json:"model"`
	CPUModel         string    `json:"cpu_model"`
	CPUSeries        string    `json:"cpu_series"`
	HDDs             []Storage `json:"hdds"`
	RAMs             []Storage `json:"rams"`
	ScreenSizeInches string    `json:"screen_size_inches"`
	CPUSpeed         string    `json:"cpu_speed"`
	GPUModel         string    `json:"gpu_model"`
	Cam              string    `json:"cam"`
	SerialNumber     string    `json:"serial_number"`
	PartNumber       string    `json:"part_number"`
	Battery          string    `json:"battery"`
	Adapter          string    `json:"adapter"`
}

type Storage struct {
	Capacity string `json:"capacity"`
	Type     string `json:"type"`
	Unit     string `json:"unit"`
}

func (h *DataHandler) Done(c *fiber.Ctx) error {
	data := []Data{}
	if err := c.BodyParser(&data); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Failed to parse body",
			"message": err.Error(),
			"body":    string(c.Body()),
		})
	}

	// convert data to excel
	excelData := excelize.NewFile()

	// set headers in row 1
	headerRow := make([]interface{}, 0)
	for _, key := range FieldKeys {
		headerRow = append(headerRow, Fields[key])
	}
	excelData.SetSheetRow("Sheet1", "A1", &headerRow)

	// set data rows starting from row 2
	rowNum := 2
	for _, row := range data {
		rowData := make([]interface{}, 0)
		for _, key := range FieldKeys {
			val := getFieldValue(row, key)
			rowData = append(rowData, val)
		}

		excelData.SetSheetRow("Sheet1", fmt.Sprintf("A%d", rowNum), &rowData)
		rowNum++
	}

	name := uuid.New().String() + ".xlsx"
	// save excel file
	err := excelData.SaveAs("files/" + name)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to save excel file",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Data added successfully",
		"link":    "files/" + name,
	})
}

func getFieldValue(data Data, key string) interface{} {
	switch key {
	case "job_number":
		return data.JobNumber
	case "type":
		return data.Type
	case "make":
		return data.Make
	case "model":
		return data.Model
	case "cpu_model":
		return data.CPUModel
	case "cpu_series":
		return data.CPUSeries
	case "serial_number":
		return data.SerialNumber
	case "part_number":
		return data.PartNumber
	case "battery":
		return data.Battery
	case "adapter":
		return data.Adapter
	case "screen_size_inches":
		return data.ScreenSizeInches
	case "rams":
		formattedRAMs := ""
		for i, ram := range data.RAMs {
			formattedRAMs += fmt.Sprintf("%s%s %s", ram.Capacity, ram.Unit, ram.Type)
			if i < len(data.RAMs)-1 {
				formattedRAMs += ":::"
			}
		}

		return formattedRAMs
	case "hdds":
		formattedHDDs := ""
		for i, hdd := range data.HDDs {
			formattedHDDs += fmt.Sprintf("%s%s %s", hdd.Capacity, hdd.Unit, hdd.Type)
			if i < len(data.HDDs)-1 {
				formattedHDDs += ":::"
			}
		}

		return formattedHDDs
	case "cpu_speed":
		return data.CPUSpeed
	case "gpu_model":
		return data.GPUModel
	case "cam":
		return data.Cam
	default:
		return ""
	}
}
