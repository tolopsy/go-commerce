package payment

import (
	"fmt"

	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/paymentintent"
)

type Config struct {
	Secret   string
	Key      string
	Currency string
}

type Transaction struct {
	StatusID       int
	Amount         int
	Currency       string
	// Last four digits of the paying credit card number
	LastFOur       string
	BankReturnCode string
}

// Charge wrapper is intended to make it easier to extend support for more payment gateways.
func (c *Config) Charge(amount int) (*stripe.PaymentIntent, string, error) {
	return c.createPaymentIntent(amount)
}

func (c *Config) createPaymentIntent(amount int) (*stripe.PaymentIntent, string, error) {
	stripe.Key = c.Secret
	var msg string

	// create payment intent
	params := &stripe.PaymentIntentParams{
		Amount: stripe.Int64(int64(amount)),
		Currency: stripe.String(c.Currency),
	}

	pi, err := paymentintent.New(params)
	if err != nil {
		if stripeErr, ok := err.(*stripe.Error); ok {
			msg = stripeCardErrorMessage(stripeErr.Code)
		}
		return nil, msg, err
	}
	return pi, msg, nil
}

func stripeCardErrorMessage(code stripe.ErrorCode) string {
	var msg string
	switch code {
	case stripe.ErrorCodeCardDeclined:
		msg = "Your card was declined"
	case stripe.ErrorCodeExpiredCard:
		msg = "Your card is expired"
	case stripe.ErrorCodeInvalidCardType:
		msg = "Your card type is not accepted"
	case stripe.ErrorCodeAmountTooLarge:
		msg = "The amount is too large to charge on your card"
	case stripe.ErrorCodeAmountTooSmall:
		msg = "The amount to charge is too small"
	case stripe.ErrorCodeIncorrectCVC:
		msg = "Incorrect CVC code"
	case stripe.ErrorCodeIncorrectZip:
		msg = "Incorrect ZIP/Postal code"
	case stripe.ErrorCodeBalanceInsufficient:
		msg = "Insufficient balance"
	case stripe.ErrorCodePostalCodeInvalid:
		msg = "Postal code is invalid"
	default:
		msg = fmt.Sprintf("Your card was declined: %s", string(code))
	}
	return msg
}