package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/phpdave11/gofpdf"
	"github.com/phpdave11/gofpdf/contrib/gofpdi"
)

type APIResponse struct {
	HasError bool   `json:"has_error"`
	Message  string `json:"message,omitempty"`
}

func (app *application) readJSON(w http.ResponseWriter, r *http.Request, data interface{}) error {
	var maxBytes int64 = 1048576
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(data)
	if err != nil {
		return err
	}

	err = decoder.Decode(&struct{}{})
	if err != io.EOF {
		fmt.Println(err.Error())
		return errors.New("request body must only have one single JSON value")
	}

	return nil
}

// writeJSON writes arbitrary data out as JSON
func (app *application) writeJSON(w http.ResponseWriter, data interface{}, statuscode int, headers ...map[string]string) error {
	out, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		return err
	}
	if len(headers) > 0 {
		for _, header := range headers {
			for k, v := range header {
				w.Header().Add(k, v)
			}
		}
	}
	w.WriteHeader(statuscode)
	w.Header().Set("Content-Type", "application/json")
	w.Write(out)
	return nil
}

func (app *application) badRequest(w http.ResponseWriter, err error) error {
	payload := APIResponse{
		HasError: true,
		Message:  err.Error(),
	}
	if err := app.writeJSON(w, payload, http.StatusBadRequest); err != nil {
		return err
	}
	return nil
}

func (app *application) GenerateInvoicePDF(data InvoiceData) error {
	pdf := gofpdf.New("P", "mm", "Letter", "")
	pdf.SetMargins(10, 13, 10)
	pdf.SetAutoPageBreak(true, 0)

	importer := gofpdi.NewImporter()
	t := importer.ImportPage(pdf, "./pdf_templates/invoice.pdf", 1, "/MediaBox")

	pdf.AddPage()
	importer.UseImportedTemplate(pdf, t, 0, 0, 215.9, 0)

	pdf.SetY(50)
	pdf.SetX(10)
	pdf.SetFont("Times", "", 11)

	pdf.CellFormat(97, 8, fmt.Sprintf("Attention: %s %s", data.FirstName, data.LastName), "", 0, "L", false, 0, "")
	pdf.Ln(5)
	pdf.CellFormat(97, 8, data.CreatedAt.Format("2006-01-02"), "", 0, "L", false, 0, "")

	pdf.SetX(58)
	pdf.SetY(93)
	pdf.CellFormat(155, 8, data.Product, "", 0, "L", false, 0, "")
	pdf.SetX(166)
	pdf.CellFormat(20, 8, fmt.Sprintf("%d", data.Quantity), "", 0, "C", false, 0, "")
	pdf.SetX(185)
	pdf.CellFormat(20, 8, fmt.Sprintf("$%.2f", float32(data.Amount/100.0)), "", 0, "R", false, 0, "")

	invoicePath := fmt.Sprintf("./invoices/%d.pdf", data.ID)
	if err := pdf.OutputFileAndClose(invoicePath); err != nil {
		return err
	}
	return nil
}

func (app *application) CreateDirIfNotExist(path string) error {
	const mode = 0755
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.Mkdir(path, mode)
		if err != nil {
			app.errorLog.Println(err)
			return err
		}
	}

	return nil
}