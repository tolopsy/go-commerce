package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

const version = "1.0.0"

type config struct {
	port   int
	smtp struct {
		host     string
		port     int
		username string
		password string
	}
	frontend  string
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

	app.infoLog.Printf(fmt.Sprintf("Starting invoice microservice on port %d", app.config.port))
	return server.ListenAndServe()
}

func main() {
	var conf config

	flag.IntVar(&conf.port, "port", 5000, "Server port to listen to (default: 9000)")
	flag.StringVar(&conf.frontend, "frontend", "http://localhost:8000", "Frontend URL")

	flag.Parse()

	conf.smtp.host = os.Getenv("SMTP_HOST")
	conf.smtp.port, _ = strconv.Atoi(os.Getenv("SMTP_PORT"))
	conf.smtp.username = os.Getenv("SMTP_USERNAME")
	conf.smtp.password = os.Getenv("SMTP_PASSWORD")

	infoLog := log.New(os.Stdout, "INFO:\t", log.Ldate|log.Ltime)
	errorLog := log.New(os.Stdout, "ERROR:\t", log.Ldate|log.Ltime|log.Lshortfile)

	app := &application{
		config:   conf,
		infoLog:  infoLog,
		errorLog: errorLog,
		version:  version,
	}

	app.CreateDirIfNotExist("./invoices")

	if err := app.serve(); err != nil {
		log.Fatalln(err)
	}
}