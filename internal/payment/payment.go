package payment

import (
	"fmt"

	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/customer"
	"github.com/stripe/stripe-go/v72/paymentintent"
	"github.com/stripe/stripe-go/v72/paymentmethod"
	"github.com/stripe/stripe-go/v72/refund"
	"github.com/stripe/stripe-go/v72/sub"
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

// Charge creates payment intent/order.
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

// GetPaymentMethod gets payment method by id
func (c *Config) GetPaymentMethod(id string) (*stripe.PaymentMethod, error) {
	stripe.Key = c.Secret

	paymentMethod, err := paymentmethod.Get(id, nil)
	if err != nil {
		return nil, err
	}
	return paymentMethod, nil
}

// RetrievePaymentIntent retrieves existing payment intent by id
func (c *Config) RetrievePaymentIntent(id string) (*stripe.PaymentIntent, error) {
	stripe.Key = c.Secret

	paymentIntent, err := paymentintent.Get(id, nil)
	if err != nil {
		return nil, err
	}

	return paymentIntent, nil
}

func (c *Config) CreateCustomer(pm, email string) (*stripe.Customer, string, error) {
	stripe.Key = c.Secret
	customerParams := &stripe.CustomerParams{
		PaymentMethod: stripe.String(pm),
		Email: stripe.String(email),
		InvoiceSettings: &stripe.CustomerInvoiceSettingsParams{
			DefaultPaymentMethod: stripe.String(pm),
		},
	}

	customer, err := customer.New(customerParams)
	if err != nil {
		msg := ""
		if stripeErr, ok := err.(*stripe.Error); ok {
			msg = stripeCardErrorMessage(stripeErr.Code)
		}
		return nil, msg, err
	}

	return customer, "", nil
}

func (c *Config) SubscribeToPlan(customer *stripe.Customer, plan, email, lastFour, cardType string) (*stripe.Subscription, error) {
	items := []*stripe.SubscriptionItemsParams{
		{Plan: stripe.String(plan)},
	}

	params := &stripe.SubscriptionParams{
		Customer: stripe.String(customer.ID),
		Items: items,
	}

	params.AddMetadata("last_four", lastFour)
	params.AddMetadata("card_type", cardType)
	params.AddExpand("latest_invoice.payment_intent")

	subscription, err := sub.New(params)
	if err != nil {
		return nil, err
	}
	return subscription, nil
}

func (c *Config) Refund(paymentIntent string, amount int) error {
	stripe.Key = c.Secret
	amountToRefund := int64(amount)
	
	refundParams := &stripe.RefundParams{
		Amount: &amountToRefund,
		PaymentIntent: &paymentIntent,
	}

	_, err := refund.New(refundParams)
	if err != nil {
		return err
	}
	return nil
}

func (c *Config) CancelSubscription(subscriptionID string) error {
	stripe.Key = c.Secret
	
	params := &stripe.SubscriptionParams{
		CancelAtPeriodEnd: stripe.Bool(true),
	}

	_, err := sub.Update(subscriptionID, params)
	if err != nil {
		return err
	}
	return nil
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