package main

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"time"

	mail "github.com/xhit/go-simple-mail/v2"
)

//go:embed email_templates
var emailTemplateFS embed.FS

func (app *application) SendMail(from, to, subject, tmpl string, attachments []string, data interface{}) error {
	templateToRender := fmt.Sprintf("email_templates/%s.html.tmpl", tmpl)
	t, err := template.New("email-html").ParseFS(emailTemplateFS, templateToRender)
	if err != nil {
		app.errorLog.Println(err)
		return err
	}

	var tpl bytes.Buffer
	if err := t.ExecuteTemplate(&tpl, "body", data); err != nil {
		app.errorLog.Println(err)
		return err
	}
	formattedMessage := tpl.String()

	templateToRender = fmt.Sprintf("email_templates/%s.plain.tmpl", tmpl)
	t, err = template.New("email-plain").ParseFS(emailTemplateFS, templateToRender)
	if err != nil {
		app.errorLog.Println(err)
		return err
	}
	if err = t.ExecuteTemplate(&tpl, "body", data); err != nil {
		app.errorLog.Println(err)
		return err
	}
	plainMessage := tpl.String()

	server := mail.NewSMTPClient()
	server.Host = app.config.smtp.host
	server.Port = app.config.smtp.port
	server.Username = app.config.smtp.username
	server.Password = app.config.smtp.password
	server.Encryption = mail.EncryptionSTARTTLS
	server.KeepAlive = false
	server.ConnectTimeout = 10 * time.Second
	server.SendTimeout = 10 * time.Second

	smtpClient, err := server.Connect()
	if err != nil {
		app.errorLog.Println(err)
		return err
	}

	email := mail.NewMSG()
	email.SetFrom(from).
		AddTo(to).
		SetSubject(subject).
		SetBody(mail.TextHTML, formattedMessage).
		AddAlternative(mail.TextPlain, plainMessage)
	
	if len(attachments) > 0 {
		for _, x := range attachments {
			email.Attach(&mail.File{Name: "", FilePath: x})
		}
	}

	if err = email.Send(smtpClient); err != nil {
		app.errorLog.Println(err)
		return err
	}

	app.infoLog.Println("sent email")
	return nil
}