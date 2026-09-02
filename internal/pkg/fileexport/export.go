// =============================================================================
// 文件: internal/pkg/fileexport/export.go
// 模块: 文件导出
// 类型: infra
// 职责: 提供表格导出的通用下载能力，支持 xlsx/xls/csv/xml/html。
// 依赖: internal/pkg/excel
// =============================================================================

package fileexport

import (
	"bytes"
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/xuri/excelize/v2"
)

// BuildFileName 构建导出文件名。
func BuildFileName(base string, fileType string, defaultPrefix string) string {
	safeBase := strings.TrimSpace(base)
	if safeBase == "" {
		prefix := strings.TrimSpace(defaultPrefix)
		if prefix == "" {
			prefix = "export"
		}
		safeBase = prefix + "_" + time.Now().Format("20060102_150405")
	}
	safeBase = strings.ReplaceAll(safeBase, "/", "_")
	safeBase = strings.ReplaceAll(safeBase, "\\", "_")
	safeBase = strings.ReplaceAll(safeBase, "\"", "_")
	safeBase = strings.ReplaceAll(safeBase, "\n", "_")
	safeBase = strings.ReplaceAll(safeBase, "\r", "_")
	return safeBase + "." + fileType
}

// WriteTable 将表格导出到响应流并触发浏览器下载。
func WriteTable(w http.ResponseWriter, fileName string, fileType string, headers []string, rows [][]string) error {
	switch fileType {
	case "xlsx":
		return writeXLSXFile(w, fileName, headers, rows)
		// return excel.Export(w, fileName, headers, toAnyRows(rows))
	case "csv":
		return writeCSVFile(w, fileName, headers, rows)
	case "xls":
		return writeXLSFile(w, fileName, headers, rows)
	case "xml":
		return writeXMLFile(w, fileName, headers, rows)
	case "html":
		return writeHTMLFile(w, fileName, headers, rows)
	default:
		return fmt.Errorf("unsupported export file type: %s", fileType)
	}
}

func runeWidth(s string) int {
	if s == "" {
		return 0
	}
	return utf8.RuneCountInString(s)
}

func autoWidth(f *excelize.File, sheet string, headers []string, rows [][]string) error {
	colCount := len(headers)
	for _, row := range rows {
		if len(row) > colCount {
			colCount = len(row)
		}
	}
	if colCount == 0 {
		return nil
	}
	widths := make([]int, colCount)
	for i, h := range headers {
		widths[i] = runeWidth(h)
	}
	for _, row := range rows {
		for i, val := range row {
			w := runeWidth(fmt.Sprint(val))
			if w > widths[i] {
				widths[i] = w
			}
		}
	}
	for i := 0; i < colCount; i++ {
		colName, err := excelize.ColumnNumberToName(i + 1)
		if err != nil {
			return err
		}
		width := float64(widths[i] + 2)
		if width < 10 {
			width = 10
		}
		if width > 60 {
			width = 60
		}
		if err := f.SetColWidth(sheet, colName, colName, width); err != nil {
			return err
		}
	}
	return nil
}

// Export 将数据写入 http.ResponseWriter，触发浏览器下载。
func writeXLSXFile(w http.ResponseWriter, filename string, headers []string, rows [][]string) error {
	if w == nil {
		return fmt.Errorf("writer is nil")
	}
	name := strings.TrimSpace(filename)
	if name == "" {
		name = "export.xlsx"
	}
	if !strings.HasSuffix(strings.ToLower(name), ".xlsx") {
		name += ".xlsx"
	}

	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	if sheet == "" {
		sheet = "Sheet1"
	}

	if len(headers) > 0 {
		for idx, h := range headers {
			cell, err := excelize.CoordinatesToCellName(idx+1, 1)
			if err != nil {
				return err
			}
			if err := f.SetCellValue(sheet, cell, h); err != nil {
				return err
			}
		}
		styleID, err := f.NewStyle(&excelize.Style{
			Font: &excelize.Font{
				Bold: true,
			},
		})
		if err != nil {
			return err
		}
		lastCell, err := excelize.CoordinatesToCellName(max(1, len(headers)), 1)
		if err != nil {
			return err
		}
		if err := f.SetCellStyle(sheet, "A1", lastCell, styleID); err != nil {
			return err
		}
	}

	for r, row := range rows {
		for c, val := range row {
			cell, err := excelize.CoordinatesToCellName(c+1, r+2)
			if err != nil {
				return err
			}
			if err := f.SetCellValue(sheet, cell, val); err != nil {
				return err
			}
		}
	}

	if err := autoWidth(f, sheet, headers, rows); err != nil {
		return err
	}

	disposition := fmt.Sprintf("attachment; filename*=UTF-8''%s", url.PathEscape(name))
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", disposition)
	w.Header().Set("Cache-Control", "no-store")
	return f.Write(w)
}

func writeCSVFile(w http.ResponseWriter, fileName string, headers []string, rows [][]string) error {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", buildAttachmentHeader(fileName))
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return err
	}
	writer := csv.NewWriter(w)
	if err := writer.Write(headers); err != nil {
		return err
	}
	for _, row := range rows {
		if err := writer.Write(row); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}

func writeXLSFile(w http.ResponseWriter, fileName string, headers []string, rows [][]string) error {
	w.Header().Set("Content-Type", "application/vnd.ms-excel; charset=utf-8")
	w.Header().Set("Content-Disposition", buildAttachmentHeader(fileName))
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)

	writer := csv.NewWriter(w)
	writer.Comma = '\t'
	if err := writer.Write(headers); err != nil {
		return err
	}
	for _, row := range rows {
		if err := writer.Write(row); err != nil {
			return err
		}
	}
	writer.Flush()
	return writer.Error()
}

func writeXMLFile(w http.ResponseWriter, fileName string, headers []string, rows [][]string) error {
	type xmlCell struct {
		Header string `xml:"header,attr"`
		Value  string `xml:",chardata"`
	}
	type xmlRow struct {
		Cells []xmlCell `xml:"cell"`
	}
	type xmlTable struct {
		XMLName xml.Name `xml:"table"`
		Rows    []xmlRow `xml:"row"`
	}

	payload := xmlTable{Rows: make([]xmlRow, 0, len(rows))}
	for _, row := range rows {
		cells := make([]xmlCell, 0, len(headers))
		for i := 0; i < len(headers) && i < len(row); i++ {
			cells = append(cells, xmlCell{
				Header: headers[i],
				Value:  row[i],
			})
		}
		payload.Rows = append(payload.Rows, xmlRow{Cells: cells})
	}
	raw, err := xml.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Content-Disposition", buildAttachmentHeader(fileName))
	w.Header().Set("Cache-Control", "no-store")
	_, err = w.Write([]byte(xml.Header + string(raw)))
	return err
}

func writeHTMLFile(w http.ResponseWriter, fileName string, headers []string, rows [][]string) error {
	var b strings.Builder
	b.WriteString("<!doctype html><html lang=\"zh-CN\"><head><meta charset=\"utf-8\"><title>导出结果</title></head><body>")
	b.WriteString("<table border=\"1\" cellspacing=\"0\" cellpadding=\"6\"><thead><tr>")
	for _, h := range headers {
		b.WriteString("<th>")
		b.WriteString(html.EscapeString(h))
		b.WriteString("</th>")
	}
	b.WriteString("</tr></thead><tbody>")
	for _, row := range rows {
		b.WriteString("<tr>")
		for _, cell := range row {
			b.WriteString("<td>")
			b.WriteString(html.EscapeString(cell))
			b.WriteString("</td>")
		}
		b.WriteString("</tr>")
	}
	b.WriteString("</tbody></table></body></html>")

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Disposition", buildAttachmentHeader(fileName))
	w.Header().Set("Cache-Control", "no-store")
	_, err := w.Write([]byte(b.String()))
	return err
}

func buildAttachmentHeader(name string) string {
	var ascii bytes.Buffer
	for _, r := range name {
		if r >= 32 && r <= 126 && r != '"' && r != '\\' {
			ascii.WriteRune(r)
			continue
		}
		ascii.WriteByte('_')
	}
	if ascii.Len() == 0 {
		ascii.WriteString("export")
	}
	return fmt.Sprintf("attachment; filename=\"%s\"; filename*=UTF-8''%s", ascii.String(), url.PathEscape(name))
}
