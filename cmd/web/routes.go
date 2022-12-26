package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (app *application) routes() http.Handler {
	mux := chi.NewRouter()
	mux.Use(SessionLoad)

	mux.Get("/", app.Home)

	mux.Route("/admin", func(r chi.Router) {
		r.Use(app.Auth)
		r.Get("/pay-terminal", app.PaymentTerminal)
		r.Get("/all-sales", app.AllSales)
		r.Get("/all-subscriptions", app.AllSubscriptions)
		r.Get("/sales/{id}", app.ShowSale)
		r.Get("/subscriptions/{id}", app.ShowSubscription)
	})
	
	// mux.Post("/terminal-payment-successful", app.TerminalPaymentSuccessful)
	// mux.Get("/terminal-receipt", app.TerminalReceipt)

	mux.Get("/widget/{id}", app.ChargeOnce)
	mux.Post("/payment-successful", app.PaymentSuccessful)
	mux.Get("/receipt", app.Receipt)

	mux.Get("/plan/bronze", app.BronzePlan)
	mux.Get("/receipt/bronze", app.BronzePlanReceipt)

	// auth routes
	mux.Get("/login", app.LoginPage)
	mux.Post("/login", app.PostLoginPage)
	mux.Get("/logout", app.Logout)

	mux.Get("/forgot-password", app.ForgotPassword)
	mux.Get("/reset-password", app.ResetPassword)

	fileServer := http.FileServer(http.Dir("./static"))
	mux.Handle("/static/*", http.StripPrefix("/static", fileServer))
	return mux
}
