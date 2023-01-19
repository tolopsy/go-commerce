package main

import (
	"fmt"
	"net/http"
	"time"
)

type InvoiceData struct {
	ID        int       `json:"id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Email     string    `json:"email"`
	Quantity  int       `json:"quantity"`
	Amount    int       `json:"amount"`
	Product   string    `json:"product"`
	CreatedAt time.Time `json:"created_at"`
}

func (app *application) CreateAndSendInvoice(w http.ResponseWriter, r *http.Request) {
	var data InvoiceData

	err := app.readJSON(w, r, &data)
	if err != nil {
		app.badRequest(w, err)
	}

	err = app.GenerateInvoicePDF(data)
	if err != nil {
		app.badRequest(w, err)
		return
	}

	attachments := []string{
		fmt.Sprintf("./invoices/%d.pdf", data.ID),
	}
	err = app.SendMail("info@widgets.com", data.Email, "Your Invoice", "invoice", attachments, nil)
	if err != nil {
		app.badRequest(w, err)
		return
	}

	// send invoice

	var resp struct {
		Error   bool `json:"error"`
		Message string `json:"message"`
	}
	resp.Error = false
	resp.Message = fmt.Sprintf("Invoice %d.pdf created and sent to %s", data.ID, data.Email)
	app.writeJSON(w, resp, http.StatusCreated)
}
