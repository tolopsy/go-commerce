package main

import (
	"net/http"
)

func (app *application) PaymentTerminal(w http.ResponseWriter, r *http.Request) {
	stringMap := make(map[string]string)
	stringMap["publishable_key"] = app.config.stripe.key
	if err := app.renderTemplate(w, r, "terminal", &templateData{StringMap: stringMap}, "stripe-js"); err != nil {
		app.errorLog.Println(err)
	}
}

func (app *application) PaymentSuccessful(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		app.errorLog.Println(err)
		return
	}

	cardHolderData := make(map[string]interface{})

	// read post data
	cardHolderData["name"] = r.Form.Get("cardholder_name")
	cardHolderData["email"] = r.Form.Get("cardholder_email")
	cardHolderData["payment_intent"] = r.Form.Get("payment_intent")
	cardHolderData["payment_method"] = r.Form.Get("payment_method")
	cardHolderData["payment_amount"] = r.Form.Get("payment_amount")
	cardHolderData["payment_currency"] = r.Form.Get("payment_currency")

	if err := app.renderTemplate(w, r, "payment_successful", &templateData{Data: cardHolderData}); err != nil {
		app.errorLog.Println(err)
	}
}
