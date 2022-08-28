package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"go-commerce/internal/driver"
)

const name = "card-pay-backend"
const version = "1.0.0"

type config struct {
	port   int
	env    string
	stripe struct {
		secret string
		key    string
	}
	db struct {
		dsn string
	}
}

type application struct {
	config   config
	infoLog  *log.Logger
	errorLog *log.Logger
	version  string
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
	var conf config

	flag.IntVar(&conf.port, "port", 9000, "Server port to listen flag on (default: 9000)")
	flag.StringVar(&conf.env, "env", "development", "Application environment (default: development) {development|staging|production}")

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

	app := &application{
		config:   conf,
		infoLog:  infoLog,
		errorLog: errorLog,
		version:  version,
	}

	if err := app.serve(); err != nil {
		log.Fatalln(err)
	}
}
