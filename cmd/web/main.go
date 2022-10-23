package main

import (
	"encoding/gob"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"go-commerce/internal/driver"
	"go-commerce/internal/models"

	"github.com/alexedwards/scs/v2"
	"github.com/alexedwards/scs/mysqlstore"
)

const name = "card-pay-web"
const version = "1.0.0"

var sessionManager *scs.SessionManager

type config struct {
	port   int
	env    string
	api    string
	stripe struct {
		secret string
		key    string
	}
	db struct {
		dsn string
	}
}

type application struct {
	config         config
	infoLog        *log.Logger
	errorLog       *log.Logger
	templateCache  map[string]*template.Template
	version        string
	DB             models.DBWrapper
	SessionManager *scs.SessionManager
}

func (app *application) serve() error {
	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", app.config.port),
		Handler:           app.routes(),
		IdleTimeout:       30 * time.Second,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      5 * time.Second,
	}

	app.infoLog.Printf(fmt.Sprintf("Starting %s server in %s mode on port %d", name, app.config.env, app.config.port))
	return server.ListenAndServe()
}

func main() {
	gob.Register(TransactionData{})
	var conf config

	flag.IntVar(&conf.port, "port", 8000, "Server port to listen flag on (default: 8000)")
	flag.StringVar(&conf.env, "env", "development", "Application environment (default: development) {development|production}")
	flag.StringVar(&conf.api, "api", "http://localhost:9000", "URL to API (default: http://localhost:9000)")

	flag.Parse()

	conf.stripe.key = os.Getenv("STRIPE_KEY")
	conf.stripe.secret = os.Getenv("STRIPE_SECRET")
	conf.db.dsn = os.Getenv("DB_DSN")

	infoLog := log.New(os.Stdout, "INFO:\t", log.Ldate|log.Ltime)
	errorLog := log.New(os.Stdout, "ERROR:\t", log.Ldate|log.Ltime|log.Lshortfile)

	conn, err := driver.OpenDB(conf.db.dsn)
	if err != nil {
		errorLog.Fatal(err)
	}
	defer conn.Close()

	// initialize session management
	sessionManager = scs.New()
	sessionManager.Lifetime = 24 * time.Hour
	sessionManager.Store = mysqlstore.New(conn)

	templateCache := make(map[string]*template.Template)

	app := &application{
		config:         conf,
		infoLog:        infoLog,
		errorLog:       errorLog,
		templateCache:  templateCache,
		version:        version,
		DB:             models.DBWrapper{DB: conn},
		SessionManager: sessionManager,
	}

	err = app.serve()
	if err != nil {
		log.Fatalln(err)
	}
}
