package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"go-commerce/internal/encryption"
	"go-commerce/internal/models"
	"go-commerce/internal/payment"
	"go-commerce/internal/urlsigner"

	"github.com/go-chi/chi/v5"
	"github.com/stripe/stripe-go/v72"
	"golang.org/x/crypto/bcrypt"
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
			PaymentIntent:       subscription.ID,
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
		Message:  fmt.Sprintf("authenticated user - %s", user.Email),
	}
	app.writeJSON(w, payload, http.StatusOK)
}

func (app *application) TerminalPaymentSuccessful(w http.ResponseWriter, r *http.Request) {
	var transactionData struct {
		FirstName       string `json:"first_name"`
		LastName        string `json:"last_name"`
		Email           string `json:"email"`
		PaymentIntentID string `json:"payment_intent"`
		PaymentMethodID string `json:"payment_method"`
		Amount          int    `json:"amount"`
		Currency        string `json:"currency"`
		LastFour        string `json:"last_four"`
		ExpiryMonth     int    `json:"expiry_month"`
		ExpiryYear      int    `json:"expiry_year"`
		BankReturnCode  string `json:"bank_return_code"`
	}
	err := app.readJSON(w, r, &transactionData)
	if err != nil {
		app.badRequest(w, err)
		return
	}

	payConf := payment.Config{
		Secret: app.config.stripe.secret,
		Key:    app.config.stripe.key,
	}
	paymentIntent, err := payConf.RetrievePaymentIntent(transactionData.PaymentIntentID)
	if err != nil {
		app.badRequest(w, err)
		return
	}
	paymentMethod, err := payConf.GetPaymentMethod(transactionData.PaymentMethodID)
	if err != nil {
		app.badRequest(w, err)
		return
	}

	transactionData.LastFour = paymentMethod.Card.Last4
	transactionData.ExpiryMonth = int(paymentMethod.Card.ExpMonth)
	transactionData.ExpiryYear = int(paymentMethod.Card.ExpYear)
	transactionData.BankReturnCode = paymentIntent.Charges.Data[0].ID

	// create new transaction
	transaction := models.Transaction{
		Amount:              transactionData.Amount,
		Currency:            transactionData.Currency,
		LastFour:            transactionData.LastFour,
		BankReturnCode:      transactionData.BankReturnCode,
		PaymentIntent:       transactionData.PaymentIntentID,
		PaymentMethod:       transactionData.PaymentMethodID,
		CardExpiryMonth:     transactionData.ExpiryMonth,
		CardExpiryYear:      transactionData.ExpiryYear,
		TransactionStatusID: 2,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}
	_, err = app.SaveTransaction(transaction)
	if err != nil {
		app.badRequest(w, err)
		return
	}

	app.writeJSON(w, transactionData, http.StatusOK)
}

func (app *application) SendPasswordResetEmail(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Email string `json:"email"`
	}

	if err := app.readJSON(w, r, &payload); err != nil {
		app.badRequest(w, err)
		return
	}

	if _, err := app.DB.GetUserByEmail(payload.Email); err != nil {
		app.badRequest(w, errors.New("no matching email found"))
		return
	}

	link := fmt.Sprintf("%s/reset-password?email=%s", app.config.frontend, payload.Email)
	signer := urlsigner.NewSigner([]byte(app.config.secretKey))

	var data struct {
		Link string
	}
	data.Link = signer.GenerateTokenFromString(link)

	// send mail
	err := app.SendMail("info@widgets.com", payload.Email, "Password Reset Email", "password_reset", data)
	if err != nil {
		app.errorLog.Println(err)
		app.badRequest(w, err)
		return
	}

	resp := APIResponse{
		HasError: false,
	}
	app.writeJSON(w, resp, http.StatusCreated)
}

func (app *application) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := app.readJSON(w, r, &payload); err != nil {
		app.badRequest(w, err)
		return
	}

	encryptor := encryption.NewEncryptor([]byte(app.config.secretKey))
	realEmail, err := encryptor.Decrypt(payload.Email)
	if err != nil {
		app.badRequest(w, err)
		return
	}

	user, err := app.DB.GetUserByEmail(realEmail)
	if err != nil {
		app.badRequest(w, err)
		return
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(payload.Password), 12)
	if err != nil {
		app.badRequest(w, err)
		return
	}

	if err = app.DB.UpdatePasswordForUser(user, string(newHash)); err != nil {
		app.badRequest(w, err)
		return
	}

	resp := APIResponse{
		HasError: false,
		Message:  "password changed",
	}
	app.writeJSON(w, resp, http.StatusCreated)
}

func (app *application) AllSales(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Page int `json:"page"`
		PageSize    int `json:"page_size"`
	}

	if err := app.readJSON(w, r, &payload); err != nil {
		app.badRequest(w, err)
		return
	}
	allSales, totalSales, lastPage, err := app.DB.GetAllSalesPaginated(payload.PageSize, payload.Page)
	if err != nil {
		app.badRequest(w, err)
		return
	}
	var paginatedSalesData struct {
		TotalSales int             `json:"total_sales"`
		LastPage   int             `json:"last_page"`
		Sales      []*models.Order `json:"sales"`
	}

	paginatedSalesData.TotalSales = totalSales
	paginatedSalesData.LastPage = lastPage
	paginatedSalesData.Sales = allSales

	app.writeJSON(w, paginatedSalesData, http.StatusOK)
}

func (app *application) AllSubscriptions(w http.ResponseWriter, r *http.Request) {
	allSales, err := app.DB.GetAllSubscriptions()
	if err != nil {
		app.badRequest(w, err)
		return
	}
	app.writeJSON(w, allSales, http.StatusOK)
}

func (app *application) GetSale(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		app.badRequest(w, err)
		return
	}
	order, err := app.DB.GetSaleByID(id)
	if err != nil {
		app.badRequest(w, err)
		return
	}
	app.writeJSON(w, order, http.StatusOK)
}

func (app *application) GetSubscription(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		app.badRequest(w, err)
		return
	}
	order, err := app.DB.GetSubscriptionByID(id)
	if err != nil {
		app.badRequest(w, err)
		return
	}
	app.writeJSON(w, order, http.StatusOK)
}

func (app *application) RefundCharge(w http.ResponseWriter, r *http.Request) {
	var chargeToRefund struct {
		ID            int    `json:"id"`
		PaymentIntent string `json:"payment_intent"`
		Amount        int    `json:"amount"`
		Currency      string `json:"currency"`
	}

	err := app.readJSON(w, r, &chargeToRefund)
	if err != nil {
		app.badRequest(w, err)
		return
	}

	payConf := payment.Config{
		Secret:   app.config.stripe.secret,
		Key:      app.config.stripe.key,
		Currency: chargeToRefund.Currency,
	}

	err = payConf.Refund(chargeToRefund.PaymentIntent, chargeToRefund.Amount)
	if err != nil {
		app.badRequest(w, err)
		return
	}

	err = app.DB.UpdateOrderStatus(chargeToRefund.ID, models.OrderRefunded)
	if err != nil {
		app.badRequest(w, errors.New("charge has been refunded but could not update in database"))
		app.errorLog.Println(err)
		return
	}

	response := APIResponse{
		HasError: false,
		Message:  "Charge refunded",
	}
	app.writeJSON(w, response, http.StatusOK)
}

func (app *application) CancelSubscription(w http.ResponseWriter, r *http.Request) {
	var subToCancel struct {
		ID            int    `json:"id"`
		PaymentIntent string `json:"payment_intent"`
		Currency      string `json:"currency"`
	}

	err := app.readJSON(w, r, &subToCancel)
	if err != nil {
		app.badRequest(w, err)
		return
	}

	payConf := payment.Config{
		Secret:   app.config.stripe.secret,
		Key:      app.config.stripe.key,
		Currency: subToCancel.Currency,
	}

	err = payConf.CancelSubscription(subToCancel.PaymentIntent)
	if err != nil {
		app.badRequest(w, err)
		return
	}

	err = app.DB.UpdateOrderStatus(subToCancel.ID, models.OrderCancelled)
	if err != nil {
		app.badRequest(w, errors.New("subscription has been canceled but could not update in database"))
		app.errorLog.Println(err)
		return
	}

	response := APIResponse{
		HasError: false,
		Message:  "Subscription Cancelled",
	}
	app.writeJSON(w, response, http.StatusOK)
}

func (app *application) AllUsers(w http.ResponseWriter, r *http.Request) {
	allUsers, err := app.DB.GetAllUsers()
	if err != nil {
		app.badRequest(w, err)
		return
	}

	app.writeJSON(w, allUsers, http.StatusOK)
}

func (app *application) OneUser(w http.ResponseWriter, r *http.Request) {
	idParam := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		app.badRequest(w, err)
		return
	}

	user, err := app.DB.GetUserById(id)
	if err != nil {
		app.badRequest(w, err)
		return
	}

	app.writeJSON(w, user, http.StatusOK)
}