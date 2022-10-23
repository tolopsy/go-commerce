package main

import (
	"go-commerce/internal/models"
	"go-commerce/internal/payment"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

type TransactionData struct {
	FirstName       string
	LastName        string
	Email           string
	PaymentIntentID string
	PaymentMethodID string
	Amount          int
	Currency        string
	LastFour        string
	ExpiryMonth     int
	ExpiryYear      int
	BankReturnCode  string
}

func (app *application) GetTransactionData(r *http.Request) (TransactionData, error) {
	var transactionData TransactionData
	if err := r.ParseForm(); err != nil {
		return transactionData, nil
	}

	firstName := r.Form.Get("first_name")
	lastName := r.Form.Get("last_name")
	email := r.Form.Get("email")

	amount, _ := strconv.Atoi(r.Form.Get("payment_amount"))
	paymentMethodId := r.Form.Get("payment_method")
	paymentIntentId := r.Form.Get("payment_intent")
	currency := r.Form.Get("payment_currency")

	payConf := payment.Config{
		Secret: app.config.stripe.secret,
		Key:    app.config.stripe.key,
	}
	paymentIntent, err := payConf.RetrievePaymentIntent(paymentIntentId)
	if err != nil {
		return transactionData, err
	}
	paymentMethod, err := payConf.GetPaymentMethod(paymentMethodId)
	if err != nil {
		return transactionData, err
	}

	lastFour := paymentMethod.Card.Last4
	expiryMonth := paymentMethod.Card.ExpMonth
	expiryYear := paymentMethod.Card.ExpYear
	bankReturnCode := paymentIntent.Charges.Data[0].ID

	transactionData = TransactionData{
		FirstName:       firstName,
		LastName:        lastName,
		Email:           email,
		PaymentIntentID: paymentIntentId,
		PaymentMethodID: paymentMethodId,
		Amount:          amount,
		Currency:        currency,
		LastFour:        lastFour,
		ExpiryMonth:     int(expiryMonth),
		ExpiryYear:      int(expiryYear),
		BankReturnCode:  bankReturnCode,
	}
	return transactionData, nil
}

func (app *application) PaymentTerminal(w http.ResponseWriter, r *http.Request) {
	stringMap := make(map[string]string)
	stringMap["publishable_key"] = app.config.stripe.key
	if err := app.renderTemplate(w, r, "terminal", &templateData{StringMap: stringMap}); err != nil {
		app.errorLog.Println(err)
	}
}

func (app *application) Home(w http.ResponseWriter, r *http.Request) {
	if err := app.renderTemplate(w, r, "home", &templateData{}); err != nil {
		app.errorLog.Println(err)
	}
}

func (app *application) ChargeOnce(w http.ResponseWriter, r *http.Request) {
	stringMap := make(map[string]string)
	stringMap["publishable_key"] = app.config.stripe.key

	id := chi.URLParam(r, "id")
	widgetID, _ := strconv.Atoi(id)

	widget, err := app.DB.GetWidget(widgetID)
	if err != nil {
		app.errorLog.Println(err) // TODO: Handle error properly
		return
	}
	data := make(map[string]interface{})
	data["widget"] = widget

	if err := app.renderTemplate(w, r, "buy", &templateData{StringMap: stringMap, Data: data}, "stripe-js"); err != nil {
		app.errorLog.Println(err)
	}
}

func (app *application) PaymentSuccessful(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		app.errorLog.Println(err)
		return
	}

	product_id, _ := strconv.Atoi(r.Form.Get("product_id"))
	trxnData, err := app.GetTransactionData(r)
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	// create new customer
	customer_id, err := app.SaveCustomer(trxnData.FirstName, trxnData.LastName, trxnData.Email)
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	// create new transaction
	transaction := models.Transaction{
		Amount:              trxnData.Amount,
		Currency:            trxnData.Currency,
		LastFour:            trxnData.LastFour,
		BankReturnCode:      trxnData.BankReturnCode,
		PaymentIntent:       trxnData.PaymentIntentID,
		PaymentMethod:       trxnData.PaymentMethodID,
		CardExpiryMonth:     trxnData.ExpiryMonth,
		CardExpiryYear:      trxnData.ExpiryYear,
		TransactionStatusID: 2,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}
	transaction_id, err := app.SaveTransaction(transaction)
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	// create new order
	order := models.Order{
		WidgetID:      product_id,
		TransactionID: transaction_id,
		CustomerID:    customer_id,
		StatusID:      1,
		Quantity:      1,
		Amount:        trxnData.Amount,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	_, err = app.SaveOrder(order)
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	// write data to session and redirect user to new page
	app.SessionManager.Put(r.Context(), "receipt", trxnData)
	http.Redirect(w, r, "/receipt", http.StatusSeeOther)
}

func (app *application) Receipt(w http.ResponseWriter, r *http.Request) {
	transactionData, ok := app.SessionManager.Get(r.Context(), "receipt").(TransactionData)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	data := make(map[string]interface{})
	data["transaction"] = transactionData
	app.SessionManager.Remove(r.Context(), "receipt")
	if err := app.renderTemplate(w, r, "receipt", &templateData{Data: data}); err != nil {
		app.errorLog.Println(err)
	}
}

func (app *application) TerminalPaymentSuccessful(w http.ResponseWriter, r *http.Request) {
	trxnData, err := app.GetTransactionData(r)
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	// create new transaction
	transaction := models.Transaction{
		Amount:              trxnData.Amount,
		Currency:            trxnData.Currency,
		LastFour:            trxnData.LastFour,
		BankReturnCode:      trxnData.BankReturnCode,
		PaymentIntent:       trxnData.PaymentIntentID,
		PaymentMethod:       trxnData.PaymentMethodID,
		CardExpiryMonth:     trxnData.ExpiryMonth,
		CardExpiryYear:      trxnData.ExpiryYear,
		TransactionStatusID: 2,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}
	_, err = app.SaveTransaction(transaction)
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	// write data to session and redirect user to new page
	app.SessionManager.Put(r.Context(), "receipt", trxnData)
	http.Redirect(w, r, "/terminal-receipt", http.StatusSeeOther)
}

func (app *application) TerminalReceipt(w http.ResponseWriter, r *http.Request) {
	transactionData, ok := app.SessionManager.Get(r.Context(), "receipt").(TransactionData)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	data := make(map[string]interface{})
	data["transaction"] = transactionData
	app.SessionManager.Remove(r.Context(), "receipt")
	if err := app.renderTemplate(w, r, "terminal_receipt", &templateData{Data: data}); err != nil {
		app.errorLog.Println(err)
	}
}

func (app *application) BronzePlan(w http.ResponseWriter, r *http.Request) {
	widget, err := app.DB.GetWidget(2)
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	data := make(map[string]interface{})
	data["widget"] = widget

	stringMap := make(map[string]string)
	stringMap["publishable_key"] = app.config.stripe.key

	if err := app.renderTemplate(w, r, "bronze_plan", &templateData{Data: data, StringMap: stringMap}); err != nil {
		app.errorLog.Println(err)
	}
}

func (app *application) BronzePlanReceipt(w http.ResponseWriter, r *http.Request) {
	if err := app.renderTemplate(w, r, "plan_receipt", &templateData{}); err != nil {
		app.errorLog.Println(err)
	}
}

// SaveCustomer saves customer and returns customer's id
func (app *application) SaveCustomer(firstName, lastName, email string) (int, error) {
	customer := models.Customer{
		FirstName: firstName,
		LastName:  lastName,
		Email:     email,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	customer_id, err := app.DB.InsertCustomer(customer)
	if err != nil {
		return 0, err
	}
	return customer_id, nil
}

// SaveTransaction saves transaction and returns its id
func (app *application) SaveTransaction(transaction models.Transaction) (int, error) {
	txn_id, err := app.DB.InsertTransaction(transaction)
	if err != nil {
		return 0, err
	}
	return txn_id, nil
}

// SaveOrder saves an order and returns its id
func (app *application) SaveOrder(order models.Order) (int, error) {
	order_id, err := app.DB.InsertOrder(order)
	if err != nil {
		return 0, err
	}
	return order_id, nil
}

func (app *application) LoginPage(w http.ResponseWriter, r *http.Request) {
	if err := app.renderTemplate(w, r, "login", &templateData{}); err != nil {
		app.errorLog.Println(err)
	}
}

func (app *application) PostLoginPage(w http.ResponseWriter, r *http.Request) {
	app.SessionManager.RenewToken(r.Context())

	if err := r.ParseForm(); err != nil {
		app.errorLog.Println(err)
		return
	}

	email := r.Form.Get("email")
	password := r.Form.Get("password")

	id, err := app.DB.Authenticate(email, password)
	if err != nil {
		app.errorLog.Println(err)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	app.SessionManager.Put(r.Context(), "userID", id)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *application) Logout(w http.ResponseWriter, r *http.Request) {
	app.SessionManager.Destroy(r.Context())
	app.SessionManager.RenewToken(r.Context())

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
