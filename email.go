package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/mailgun/mailgun-go/v4"
	"gorm.io/gorm"
	"html/template"
	"log"
	"net/http"
	"time"
)

type Email struct {
	gorm.Model
	Email string `gorm:"unique"`
}

type EmailForm struct {
	Email string
}

func handleRegisterEmail(ctx ServerContext) func(*gin.Context) {
	return func(c *gin.Context) {
		var email EmailForm
		c.BindJSON(&email)
		err := saveEmail(ctx, email.Email)
		if err != nil {
			c.Error(err)
		}
		sendDoggoEmail(ctx, email.Email)
		c.JSON(http.StatusOK, gin.H{
			"status": "email registered!",
		})
	}
}

func saveEmail(ctx ServerContext, email string) error {
	e := Email{
		Email: email,
	}
	err := ctx.gdb.Create(&e).Error
	return err
}

func getAllEmails(ctx ServerContext) ([]Email, error) {
	var emails []Email
	err := ctx.gdb.Find(&emails).Error
	return emails, err
}

func sendMail(mg *mailgun.MailgunImpl, recipient string, doggos DoggoStatus) {

	sender := "doggos@jams.dog"
	subject := "Doggos!"

	// The message object allows you to add attachments and Bcc recipients
	message := mg.NewMessage(sender, subject, "", recipient)

	tmpl, err := template.ParseGlob(templatePath)
	if err != nil {
		log.Panicf("Could not load doggo template with error %v", err)
	}

	var emailHTML bytes.Buffer
	err = tmpl.Execute(
		&emailHTML,
		gin.H{
			"title":  "SFSPCA Doggos",
			"doggos": doggos,
		})
	message.SetHtml(emailHTML.String())

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Send the message with a 10 second timeout
	resp, id, err := mg.Send(ctx, message)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("ID: %s Resp: %s\n", id, resp)
}

func sendDoggoEmail(ctx ServerContext, recipient string) error {
	doggos, err := fetchDBDoggos(ctx)
	sendMail(ctx.mg, recipient, doggos)
	return err
}
