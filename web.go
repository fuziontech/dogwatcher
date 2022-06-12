package main

import (
	"github.com/gin-gonic/autotls"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
)

func startWebServer(ctx ServerContext, webDomain string) {
	r := gin.Default()
	r.LoadHTMLGlob(templatePath)
	r.Static("/static", "./static")
	r.StaticFile("/favicon.ico", "./static/favicon.ico")
	r.GET("/", func(c *gin.Context) {
		doggos, err := fetchDBDoggos(ctx)
		if err != nil {
			c.Error(err)
		}

		c.HTML(http.StatusOK, "doggos.html", gin.H{
			"title":  "SFSPCA Doggos",
			"doggos": doggos,
		})
	})
	r.POST("/email/signup", handleRegisterEmail(ctx))
	r.GET("/emails/send", func(c *gin.Context) {
		doggos, err := fetchDBDoggos(ctx)
		if err != nil {
			c.Error(err)
		}

		recipients, err := getAllEmails(ctx)
		if err != nil {
			c.Error(err)
		}
		for _, r := range recipients {
			sendMail(ctx.mg, r, doggos)
		}
		c.String(http.StatusOK, "Sent!")
	})
	if ctx.isProduction {
		log.Fatal(autotls.Run(r, webDomain))
	} else {
		r.Run()
	}
}
