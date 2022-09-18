package main

import (
	"encoding/json"
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

type APIResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
	Content string `json:"content,omitempty"`
	ID      string `json:"id,omitempty"`
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
	okay := true
	var subscription *stripe.Subscription
	transactionMsg := "Transaction Successful"

	stripeCustomer, msg, err := payConf.CreateCustomer(payload.PaymentMethod, payload.Email)
	if err != nil {
		app.errorLog.Println(err)
		okay = false
		transactionMsg = msg
	}

	if okay {
		subscription, err = payConf.SubscribeToPlan(stripeCustomer, payload.Plan, payload.Email, payload.LastFour, "")
		if err != nil {
			app.errorLog.Println(err)
			okay = false
			transactionMsg = "Error subscribing customer to plan"
		}
	}

	if okay {
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
		OK:      okay,
		Message: transactionMsg,
	}

	out, err := json.MarshalIndent(resp, "", "	")
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(out)
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
