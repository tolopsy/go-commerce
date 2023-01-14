package models

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
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
	OrderCleared   = 1
	OrderRefunded  = 2
	OrderCancelled = 3
)

// Order is the type for orders
type Order struct {
	ID            int         `json:"id"`
	WidgetID      int         `json:"widget_id"`
	TransactionID int         `json:"transaction_id"`
	CustomerID    int         `json:"customer_id"`
	StatusID      int         `json:"status_id"`
	Quantity      int         `json:"quantity"`
	Amount        int         `json:"amount"`
	CreatedAt     time.Time   `json:"-"`
	UpdatedAt     time.Time   `json:"-"`
	Widget        Widget      `json:"widget"`
	Transaction   Transaction `json:"transaction"`
	Customer      Customer    `json:"customer"`
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

func (w *DBWrapper) Authenticate(email, password string) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var id int
	var hashedPassword string

	row := w.DB.QueryRowContext(ctx, "select id, password from users where email = ?", email)
	err := row.Scan(&id, &hashedPassword)
	if err != nil {
		return 0, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err == bcrypt.ErrMismatchedHashAndPassword {
		return 0, errors.New("incorrect password")
	} else if err != nil {
		return 0, err
	}

	return id, nil
}

func (w *DBWrapper) UpdatePasswordForUser(u User, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err = w.DB.ExecContext(ctx, `update users set password = ? where id = ?`, string(hash), u.ID)
	if err != nil {
		return err
	}
	return nil
}

func (m *DBWrapper) GetAllSales() ([]*Order, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var orders []*Order

	query := `
	select
		o.id, o.widget_id, o.transaction_id, o.customer_id, 
		o.status_id, o.quantity, o.amount, o.created_at,
		o.updated_at, w.id, w.name, t.id, t.amount, t.currency,
		t.last_four, t.expiry_month, t.expiry_year, t.payment_intent,
		t.bank_return_code, c.id, c.first_name, c.last_name, c.email
		
	from
		orders o
		left join widgets w on (o.widget_id = w.id)
		left join transactions t on (o.transaction_id = t.id)
		left join customers c on (o.customer_id = c.id)
	where
		w.is_recurring = 0
	order by
		o.created_at desc
	`

	rows, err := m.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var o Order
		err = rows.Scan(
			&o.ID,
			&o.WidgetID,
			&o.TransactionID,
			&o.CustomerID,
			&o.StatusID,
			&o.Quantity,
			&o.Amount,
			&o.CreatedAt,
			&o.UpdatedAt,
			&o.Widget.ID,
			&o.Widget.Name,
			&o.Transaction.ID,
			&o.Transaction.Amount,
			&o.Transaction.Currency,
			&o.Transaction.LastFour,
			&o.Transaction.CardExpiryMonth,
			&o.Transaction.CardExpiryYear,
			&o.Transaction.PaymentIntent,
			&o.Transaction.BankReturnCode,
			&o.Customer.ID,
			&o.Customer.FirstName,
			&o.Customer.LastName,
			&o.Customer.Email,
		)
		if err != nil {
			return nil, err
		}
		orders = append(orders, &o)
	}

	return orders, nil
}

func (m *DBWrapper) GetAllSalesPaginated(pageSize, page int) ([]*Order, int, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	offset := (page - 1) * pageSize

	var orders []*Order

	query := `
	select
		o.id, o.widget_id, o.transaction_id, o.customer_id, 
		o.status_id, o.quantity, o.amount, o.created_at,
		o.updated_at, w.id, w.name, t.id, t.amount, t.currency,
		t.last_four, t.expiry_month, t.expiry_year, t.payment_intent,
		t.bank_return_code, c.id, c.first_name, c.last_name, c.email
		
	from
		orders o
		left join widgets w on (o.widget_id = w.id)
		left join transactions t on (o.transaction_id = t.id)
		left join customers c on (o.customer_id = c.id)
	where
		w.is_recurring = 0
	order by
		o.created_at desc
	limit ? offset ?
	`

	rows, err := m.DB.QueryContext(ctx, query, pageSize, offset)
	if err != nil {
		return nil, 0, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var o Order
		err = rows.Scan(
			&o.ID,
			&o.WidgetID,
			&o.TransactionID,
			&o.CustomerID,
			&o.StatusID,
			&o.Quantity,
			&o.Amount,
			&o.CreatedAt,
			&o.UpdatedAt,
			&o.Widget.ID,
			&o.Widget.Name,
			&o.Transaction.ID,
			&o.Transaction.Amount,
			&o.Transaction.Currency,
			&o.Transaction.LastFour,
			&o.Transaction.CardExpiryMonth,
			&o.Transaction.CardExpiryYear,
			&o.Transaction.PaymentIntent,
			&o.Transaction.BankReturnCode,
			&o.Customer.ID,
			&o.Customer.FirstName,
			&o.Customer.LastName,
			&o.Customer.Email,
		)
		if err != nil {
			return nil, 0, 0, err
		}
		orders = append(orders, &o)
	}

	query = `
		select count(o.id) from orders o
		left join widgets w on (o.widget_id = w.id)
		where w.is_recurring = 0
	`

	var totalSales int
	countSales := m.DB.QueryRowContext(ctx, query)
	err = countSales.Scan(&totalSales)
	if err != nil {
		return nil, 0, 0, err
	}

	lastPage := int(math.Ceil(float64(totalSales) / float64(pageSize)))

	return orders, totalSales, lastPage, nil
}

func (m *DBWrapper) GetAllSubscriptions() ([]*Order, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var orders []*Order

	query := `
	select
		o.id, o.widget_id, o.transaction_id, o.customer_id, 
		o.status_id, o.quantity, o.amount, o.created_at,
		o.updated_at, w.id, w.name, t.id, t.amount, t.currency,
		t.last_four, t.expiry_month, t.expiry_year, t.payment_intent,
		t.bank_return_code, c.id, c.first_name, c.last_name, c.email
		
	from
		orders o
		left join widgets w on (o.widget_id = w.id)
		left join transactions t on (o.transaction_id = t.id)
		left join customers c on (o.customer_id = c.id)
	where
		w.is_recurring = 1
	order by
		o.created_at desc
	`

	rows, err := m.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var o Order
		err = rows.Scan(
			&o.ID,
			&o.WidgetID,
			&o.TransactionID,
			&o.CustomerID,
			&o.StatusID,
			&o.Quantity,
			&o.Amount,
			&o.CreatedAt,
			&o.UpdatedAt,
			&o.Widget.ID,
			&o.Widget.Name,
			&o.Transaction.ID,
			&o.Transaction.Amount,
			&o.Transaction.Currency,
			&o.Transaction.LastFour,
			&o.Transaction.CardExpiryMonth,
			&o.Transaction.CardExpiryYear,
			&o.Transaction.PaymentIntent,
			&o.Transaction.BankReturnCode,
			&o.Customer.ID,
			&o.Customer.FirstName,
			&o.Customer.LastName,
			&o.Customer.Email,
		)
		if err != nil {
			return nil, err
		}
		orders = append(orders, &o)
	}

	return orders, nil
}

func (m *DBWrapper) GetSaleByID(id int) (Order, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
	select
		o.id, o.widget_id, o.transaction_id, o.customer_id, 
		o.status_id, o.quantity, o.amount, o.created_at,
		o.updated_at, w.id, w.name, t.id, t.amount, t.currency,
		t.last_four, t.expiry_month, t.expiry_year, t.payment_intent,
		t.bank_return_code, c.id, c.first_name, c.last_name, c.email
		
	from
		orders o
		left join widgets w on (o.widget_id = w.id)
		left join transactions t on (o.transaction_id = t.id)
		left join customers c on (o.customer_id = c.id)
	where
		o.id = ? and w.is_recurring = 0
	`

	row := m.DB.QueryRowContext(ctx, query, id)

	var o Order
	err := row.Scan(
		&o.ID,
		&o.WidgetID,
		&o.TransactionID,
		&o.CustomerID,
		&o.StatusID,
		&o.Quantity,
		&o.Amount,
		&o.CreatedAt,
		&o.UpdatedAt,
		&o.Widget.ID,
		&o.Widget.Name,
		&o.Transaction.ID,
		&o.Transaction.Amount,
		&o.Transaction.Currency,
		&o.Transaction.LastFour,
		&o.Transaction.CardExpiryMonth,
		&o.Transaction.CardExpiryYear,
		&o.Transaction.PaymentIntent,
		&o.Transaction.BankReturnCode,
		&o.Customer.ID,
		&o.Customer.FirstName,
		&o.Customer.LastName,
		&o.Customer.Email,
	)
	if err != nil {
		return o, err
	}

	return o, nil
}

func (m *DBWrapper) GetSubscriptionByID(id int) (Order, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	query := `
	select
		o.id, o.widget_id, o.transaction_id, o.customer_id, 
		o.status_id, o.quantity, o.amount, o.created_at,
		o.updated_at, w.id, w.name, t.id, t.amount, t.currency,
		t.last_four, t.expiry_month, t.expiry_year, t.payment_intent,
		t.bank_return_code, c.id, c.first_name, c.last_name, c.email
		
	from
		orders o
		left join widgets w on (o.widget_id = w.id)
		left join transactions t on (o.transaction_id = t.id)
		left join customers c on (o.customer_id = c.id)
	where
		o.id = ? and w.is_recurring = 1
	`

	row := m.DB.QueryRowContext(ctx, query, id)

	var o Order
	err := row.Scan(
		&o.ID,
		&o.WidgetID,
		&o.TransactionID,
		&o.CustomerID,
		&o.StatusID,
		&o.Quantity,
		&o.Amount,
		&o.CreatedAt,
		&o.UpdatedAt,
		&o.Widget.ID,
		&o.Widget.Name,
		&o.Transaction.ID,
		&o.Transaction.Amount,
		&o.Transaction.Currency,
		&o.Transaction.LastFour,
		&o.Transaction.CardExpiryMonth,
		&o.Transaction.CardExpiryYear,
		&o.Transaction.PaymentIntent,
		&o.Transaction.BankReturnCode,
		&o.Customer.ID,
		&o.Customer.FirstName,
		&o.Customer.LastName,
		&o.Customer.Email,
	)
	if err != nil {
		return o, err
	}

	return o, nil
}


func (m *DBWrapper) UpdateOrderStatus(id, statusID int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	statement := "update orders set status_id = ? where id = ?"
	_, err := m.DB.ExecContext(ctx, statement, statusID, id)
	if err != nil {
		return err
	}

	return nil
}

func (m *DBWrapper) GetAllUsers() ([]*User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	statement := `
		select id, first_name, last_name, email, created_at, updated_at
		from users
		order by last_name, first_name
	`

	rows, err := m.DB.QueryContext(ctx, statement)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var users []*User
	for rows.Next(){
		var u User
		err = rows.Scan(&u.ID, &u.FirstName, &u.LastName, &u.Email, &u.CreatedAt, &u.UpdatedAt)
		if err != nil {
			return nil, err
		}
		users = append(users, &u)
	}
	return users, nil
}

func (m *DBWrapper) GetUserById(id int) (User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	statement := `
		select id, first_name, last_name, email, created_at, updated_at
		from users
		where id = ?
	`

	var u User
	row := m.DB.QueryRowContext(ctx, statement, id)
	err := row.Scan(&u.ID, &u.FirstName, &u.LastName, &u.Email, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return u, err
	}
	return u, nil
}

func (m *DBWrapper) EditUser(u User) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	statement := `
		update users set
			first_name = ?,
			last_name = ?,
			email = ?,
			updated_at = ?
		where id = ?
	`
	_, err := m.DB.ExecContext(ctx, statement, u.FirstName, u.LastName, u.Email, time.Now(), u.ID)
	if err != nil {
		return err
	}
	return nil
}

func (m *DBWrapper) AddUser(u User, hash string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	statement := `
		insert into users (first_name, last_name, email, password, created_at, updated_at)
		values (?, ?, ?, ?, ?, ?)
	`

	_, err := m.DB.ExecContext(ctx, statement,
		u.FirstName,
		u.LastName,
		u.Email,
		hash,
		time.Now(),
		time.Now(),
	)

	if err != nil {
		return err
	}
	return nil
}

func (m *DBWrapper) DeleteUser(id int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	statement := `delete from users where id = ?`
	_, err := m.DB.ExecContext(ctx, statement, id)
	if err != nil {
		return err
	}

	return nil
}