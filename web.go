package main

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func startWebServer(ctx ServerContext) {
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
	r.GET("/emails/send", func(c *gin.Context) {
		doggos, err := fetchDBDoggos(ctx)
		if err != nil {
			c.Error(err)
		}

		for _, r := range ctx.recipients {
			sendMail(ctx.mg, r, doggos)
		}
		c.String(http.StatusOK, "Sent!")
	})
	r.Run()
}
