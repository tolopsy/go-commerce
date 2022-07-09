
# CardPay
CardPay is a simple demonstration of how to integrate payment with stripe using Go.

## Pre-requisite
- Ensure you have the make utility installed.
- Replace `STRIPE_KEY` and `STRIPE_SECRET` in the Makefile with your stripe publishable key and stripe secret key respectively.

## Usage
- To run both the backend and the frontend, Run `make start`
- To stop running both backend and frontend, run `make stop`
- You can find other useful commands in the [Makefile](https://github.com/tolopsy/card-pay/blob/main/Makefile)
- By default, the frontend (cardpay_web) is served in localhost:8000 while the backend (cardpay_api) is served in localhost:9000. You can change these by assigning different ports to `WEB_PORT` and `API_PORT`
- The payment page is served at `localhost:8000/pay` and the receipt page (that signifies successful payment) is served at `localhost:8000/payment-successful` route.


Credit: [Trevor Sawler](https://github.com/tsawler)