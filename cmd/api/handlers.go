package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"go-commerce/internal/models"
	"go-commerce/internal/payment"

	"github.com/go-chi/chi/v5"
	"github.com/stripe/stripe-go/v72"
)

type ChargeRequestPayload struct {
	Currency      string `json:"currency"`
	Amount        string `json:"amount"`
	PaymentMethod string `json:"payment_method"`
	Email         string `json:"email"`
	LastFour      string `json:"last_four"`
	Plan          string `json:"plan"`
	CardBrand     string `json:"card_brand"`
	ExpiryMonth   int    `json:"exp_month"`
	ExpiryYear    int    `json:"exp_year"`
	ProductID     string `json:"product_id"`
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
}

func (app *application) GetPaymentIntent(w http.ResponseWriter, r *http.Request) {
	var payload ChargeRequestPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		app.errorLog.Println(err)
		return
	}

	amount, err := strconv.Atoi(payload.Amount)
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	payConf := payment.Config{
		Secret:   app.config.stripe.secret,
		Key:      app.config.stripe.key,
		Currency: payload.Currency,
	}
	okay := true
	paymentIntent, msg, err := payConf.Charge(amount)
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
			HasError: true,
			Message:  msg,
		}
		out, err = json.MarshalIndent(errorResponse, "", "    ")
		if err != nil {
			app.errorLog.Println(err)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(out)
}

func (app *application) GetWidgetById(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	widgetID, _ := strconv.Atoi(id)

	widget, err := app.DB.GetWidget(widgetID)
	if err != nil {
		app.errorLog.Println(err)
		return
	}
	out, err := json.MarshalIndent(widget, "", "	")
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(out)
}

func (app *application) CreateCustomerAndSubscribeToPlan(w http.ResponseWriter, r *http.Request) {
	var payload ChargeRequestPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		app.errorLog.Println(err)
		return
	}

	payConf := payment.Config{
		Secret:   app.config.stripe.secret,
		Key:      app.config.stripe.key,
		Currency: payload.Currency,
	}

	var subscription *stripe.Subscription

	hasError := false
	transactionMsg := "Transaction Successful"

	stripeCustomer, msg, err := payConf.CreateCustomer(payload.PaymentMethod, payload.Email)
	if err != nil {
		app.errorLog.Println(err)
		hasError = true
		transactionMsg = msg
	}

	if !hasError {
		subscription, err = payConf.SubscribeToPlan(stripeCustomer, payload.Plan, payload.Email, payload.LastFour, "")
		if err != nil {
			app.errorLog.Println(err)
			hasError = true
			transactionMsg = "Error subscribing customer to plan"
		}
	}

	if !hasError {
		app.infoLog.Println("New subscriber with ID: ", subscription.ID)
		// store customer, order, transaction
		customerID, err := app.SaveCustomer(payload.FirstName, payload.LastName, payload.Email)
		if err != nil {
			app.errorLog.Println(err)
			return
		}

		amount, _ := strconv.Atoi(payload.Amount)
		transaction := models.Transaction{
			Amount:              amount,
			Currency:            payload.Currency,
			LastFour:            payload.LastFour,
			CardExpiryMonth:     payload.ExpiryMonth,
			CardExpiryYear:      payload.ExpiryYear,
			PaymentMethod:       payload.PaymentMethod,
			TransactionStatusID: models.TransactionCleared,
			CreatedAt:           time.Now(),
			UpdatedAt:           time.Now(),
		}
		transactionID, err := app.SaveTransaction(transaction)
		if err != nil {
			app.errorLog.Println(err)
			return
		}

		productID, _ := strconv.Atoi(payload.ProductID)
		order := models.Order{
			WidgetID:      productID,
			CustomerID:    customerID,
			TransactionID: transactionID,
			Quantity:      1,
			Amount:        amount,
			StatusID:      models.OrderCleared,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}
		_, err = app.SaveOrder(order)
		if err != nil {
			app.errorLog.Println(err)
			return
		}
	}

	resp := APIResponse{
		HasError: hasError,
		Message:  transactionMsg,
	}

	out, err := json.MarshalIndent(resp, "", "	")
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(out)
}

func (app *application) CreateAuthToken(w http.ResponseWriter, r *http.Request) {
	var userInput struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := app.readJSON(w, r, &userInput)

	if err != nil {
		app.badRequest(w, err)
		return
	}

	user, err := app.DB.GetUserByEmail(userInput.Email)
	if err != nil {
		app.invalidCredentials(w)
		return
	}

	isValidPassword, err := app.passwordMatches(user.Password, userInput.Password)
	if err != nil {
		app.invalidCredentials(w)
		return
	}

	if !isValidPassword {
		app.invalidCredentials(w)
		return
	}

	token, err := models.GenerateToken(user.ID, 24*time.Hour, models.ScopeAuthentication)
	if err != nil {
		app.badRequest(w, err)
		return
	}

	err = app.DB.InsertToken(token, user)
	if err != nil {
		app.badRequest(w, err)
		return
	}

	var payload struct {
		HasError bool          `json:"has_error"`
		Message  string        `json:"message"`
		Token    *models.Token `json:"authentication_token"`
	}

	payload.HasError = false
	payload.Message = fmt.Sprintf("Token for %s created", user.Email)
	payload.Token = token
	app.writeJSON(w, payload, http.StatusOK)
}

func (app *application) CheckAuthentication(w http.ResponseWriter, r *http.Request) {
	user, err := app.authenticateToken(r)
	if err != nil {
		app.invalidCredentials(w)
		return
	}
	payload := APIResponse{
		HasError: false,
		Message: fmt.Sprintf("authenticated user - %s", user.Email),
	}
	app.writeJSON(w, payload, http.StatusOK)
}
