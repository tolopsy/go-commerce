package main

import (
	"bytes"
	"encoding/json"
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

func (app *application) CallInvoiceMicroService(data InvoiceData) error {
	url := "http://localhost:5000/create-and-send"
	out, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(out))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	app.infoLog.Println(resp.Body)
	return nil
}