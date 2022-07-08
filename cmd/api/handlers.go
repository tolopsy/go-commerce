package main

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/tolopsy/card-pay/internal/payment"
)

type ChargeRequestPayload struct {
	Currency string `json:"currency"`
	Amount   string `json:"amount"`
}

type APIResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
	Content string `json:"content,omitempty"`
	ID      string `json:"id,omitempty"`
}

func (app *application) GetPaymentIntent(w http.ResponseWriter, r *http.Request) {
	var payload ChargeRequestPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		app.errorLog.Println(err) // TODO: Handle error properly
		return
	}

	amount, err := strconv.Atoi(payload.Amount)
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	card := payment.Config{
		Secret:   app.config.stripe.secret,
		Key:      app.config.stripe.key,
		Currency: payload.Currency,
	}
	okay := true
	paymentIntent, msg, err := card.Charge(amount)
	if err != nil {
		okay = false
	}

	var out []byte
	if okay {
		out, err = json.MarshalIndent(paymentIntent, "", "    ")
		if err != nil {
			app.errorLog.Println(err)
			return
		}
	} else {
		errorResponse := APIResponse{
			OK:      false,
			Message: msg,
			Content: "",
		}
		out, err = json.MarshalIndent(errorResponse, "", "    ")
		if err != nil {
			app.errorLog.Println(err)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(out)
}
