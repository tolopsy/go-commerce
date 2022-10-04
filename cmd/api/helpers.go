package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"go-commerce/internal/models"

	"golang.org/x/crypto/bcrypt"
)

type APIResponse struct {
	HasError bool `json:"has_error"`
	Message string `json:"message,omitempty"`
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

func (app *application) invalidCredentials(w http.ResponseWriter) error {
	payload := APIResponse{
		HasError: true,
		Message: "invalid authentication credentials",
	}
	if err := app.writeJSON(w, payload, http.StatusUnauthorized); err != nil {
		return err
	}
	return nil
}

func (app *application) passwordMatches(hash, password string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		switch {
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			return false, nil
		default:
			return false, err
		}
	}

	return true, nil
}

func (app *application) badRequest(w http.ResponseWriter, err error) error {
	payload := APIResponse{
		HasError: true,
		Message: err.Error(),
	}
	if err := app.writeJSON(w, payload, http.StatusBadRequest); err != nil {
		return err
	}
	return nil
}