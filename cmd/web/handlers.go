package main

import "net/http"

func (app *application) PaymentTerminal(w http.ResponseWriter, r *http.Request) {
	stringMap := make(map[string]string)
	stringMap["publishable_key"] = app.config.stripe.key
	if err := app.renderTemplate(w, r, "terminal", &templateData{StringMap: stringMap}); err != nil {
		app.errorLog.Println(err)
	}
}

func (app *application) PaymentSuccessful(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		app.errorLog.Println(err)
		return
	}

	// read post data
	cardHolderName := r.Form.Get("cardholder_name")
	cardHolderEmail := r.Form.Get("cardholder_email")
	paymentIntent := r.Form.Get("payment_intent")
	paymentMethod := r.Form.Get("payment_method")
	paymentAmount := r.Form.Get("payment_amount")
	paymentCurrency := r.Form.Get("payment_currency")

	cardHolderData := make(map[string]interface{})
	cardHolderData["name"] = cardHolderName
	cardHolderData["email"] = cardHolderEmail
	cardHolderData["payment_intent"] = paymentIntent
	cardHolderData["payment_method"] = paymentMethod
	cardHolderData["payment_amount"] = paymentAmount
	cardHolderData["payment_currency"] = paymentCurrency

	if err := app.renderTemplate(w, r, "payment_successful", &templateData{Data: cardHolderData}); err != nil {
		app.errorLog.Println(err)
	}
}