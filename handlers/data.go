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
	"ram_capacity_size",
	"screen_size_inches",
	"hdd_capacity",
	"hdd_type",
	"ram_type",
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
	"ram_capacity_size":  "RAM Capacity Size",
	"screen_size_inches": "Screen Size Inches",
	"hdd_capacity":       "HDD Capacity",
	"hdd_type":           "HDD Type",
	"ram_type":           "RAM Type",
	"cpu_speed":          "CPU Speed",
	"gpu_model":          "GPU Model",
	"cam":                "Cam",
}

func (h *DataHandler) Done(c *fiber.Ctx) error {
	data := make([]map[string]interface{}, 0)
	if err := c.BodyParser(&data); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Failed to parse body",
			"message": err.Error(),
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
			if val, ok := row[key]; ok {
				rowData = append(rowData, val)
			} else {
				rowData = append(rowData, "")
			}
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
