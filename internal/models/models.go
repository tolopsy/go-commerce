package models

import (
	"context"
	"database/sql"
	"strings"
	"time"
)

// DBWrapper is the type for database connection
type DBWrapper struct {
	DB *sql.DB
}

// Models is the wrapper for all models
type Models struct {
	DB DBWrapper
}

func NewModels(db *sql.DB) Models {
	return Models{
		DB: DBWrapper{DB: db},
	}
}

// Widget is the type for widgets
type Widget struct {
	ID             int       `json:"id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	InventoryLevel int       `json:"inventory_level"`
	Price          int       `json:"price"`
	Image          string    `json:"image"`
	IsRecurring    bool      `json:"is_recurring"`
	PlanID         string    `json:"plan_id"`
	CreatedAt      time.Time `json:"-"`
	UpdatedAt      time.Time `json:"-"`
}

const (
	OrderCleared = 1
	OrderRefunded = 1
	OrderCancelled = 1
)
// Order is the type for orders
type Order struct {
	ID            int       `json:"id"`
	WidgetID      int       `json:"widget_id"`
	TransactionID int       `json:"transaction_id"`
	CustomerID    int       `json:"customer_id"`
	StatusID      int       `json:"status_id"`
	Quantity      int       `json:"quantity"`
	Amount        int       `json:"amount"`
	CreatedAt     time.Time `json:"-"`
	UpdatedAt     time.Time `json:"-"`
}

// Status is the type for statuses
type Status struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}

// TransactionStatus is the type for transaction statuses
type TransactionStatus struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}

// Transaction Status Options
const (
	TransactionPending           = 1
	TransactionCleared           = 2
	TransactionDeclined          = 3
	TransactionRefunded          = 4
	TransactionPartiallyRefunded = 5
)

// Transaction is the type for transactions
type Transaction struct {
	ID                  int       `json:"id"`
	Amount              int       `json:"amount"`
	Currency            string    `json:"currency"`
	LastFour            string    `json:"last_four"`
	BankReturnCode      string    `json:"bank_return_code"`
	CardExpiryMonth     int       `json:"expiry_month"`
	CardExpiryYear      int       `json:"expiry_year"`
	PaymentIntent       string    `json:"payment_intent"`
	PaymentMethod       string    `json:"payment_method"`
	TransactionStatusID int       `json:"transaction_status_id"`
	CreatedAt           time.Time `json:"-"`
	UpdatedAt           time.Time `json:"-"`
}

// User is the type for users
type User struct {
	ID        int       `json:"id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Password  string    `json:"password"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}

// Customer is the type for customers
type Customer struct {
	ID        int       `json:"id"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}

func (w *DBWrapper) GetWidget(id int) (Widget, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var widget Widget
	row := w.DB.QueryRowContext(ctx, `
		select
			id, name, description, inventory_level, price,
			coalesce(image, ''), is_recurring, plan_id, created_at, updated_at
		from widgets
		where id = ?`, id)
	if err := row.Scan(
		&widget.ID,
		&widget.Name,
		&widget.Description,
		&widget.InventoryLevel,
		&widget.Price,
		&widget.Image,
		&widget.IsRecurring,
		&widget.PlanID,
		&widget.CreatedAt,
		&widget.UpdatedAt,
	); err != nil {
		return widget, err
	}
	return widget, nil
}

// InsertTransaction inserts a new transaction and returns its ID.
func (w *DBWrapper) InsertTransaction(txn Transaction) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	statement := `
		insert into transactions
			(amount, currency, last_four, bank_return_code, payment_intent, payment_method, transaction_status_id, expiry_month, expiry_year, created_at, updated_at)
			values (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := w.DB.ExecContext(ctx, statement,
		txn.Amount,
		txn.Currency,
		txn.LastFour,
		txn.BankReturnCode,
		txn.PaymentIntent,
		txn.PaymentMethod,
		txn.TransactionStatusID,
		txn.CardExpiryMonth,
		txn.CardExpiryYear,
		txn.CreatedAt,
		txn.UpdatedAt,
	)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

// InsertOrder inserts a new order and returns its ID.
func (w *DBWrapper) InsertOrder(order Order) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	statement := `
		insert into orders
			(widget_id, customer_id, transaction_id, status_id, quantity, amount, created_at, updated_at)
			values (?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := w.DB.ExecContext(ctx, statement,
		order.WidgetID,
		order.CustomerID,
		order.TransactionID,
		order.StatusID,
		order.Quantity,
		order.Amount,
		order.CreatedAt,
		order.UpdatedAt,
	)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

// InsertCustomer inserts a new customer and returns its ID.
func (w *DBWrapper) InsertCustomer(customer Customer) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	statement := `
		insert into customers
			(first_name, last_name, email, created_at, updated_at)
			values (?, ?, ?, ?, ?)
	`

	result, err := w.DB.ExecContext(ctx, statement,
		customer.FirstName,
		customer.LastName,
		customer.Email,
		customer.CreatedAt,
		customer.UpdatedAt,
	)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

// GetUserByEmail gets user by email address
func (w *DBWrapper) GetUserByEmail(email string) (User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var user User
	row := w.DB.QueryRowContext(ctx, `
		select
			id, first_name, last_name, email, password, created_at, updated_at
		from users
		where email = ?`, strings.ToLower(email))
	if err := row.Scan(
		&user.ID,
		&user.FirstName,
		&user.LastName,
		&user.Email,
		&user.Password,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		return user, err
	}
	return user, nil
}
