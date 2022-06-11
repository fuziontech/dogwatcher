package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/mailgun/mailgun-go/v4"
	"log"
	"text/template"
	"time"
)

func sendMail(mg *mailgun.MailgunImpl, recipient string, doggos SFSPCA_Response) {

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
