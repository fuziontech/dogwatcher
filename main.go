package main

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/go-co-op/gocron"
	"github.com/mailgun/mailgun-go/v4"
	"github.com/spf13/viper"
	"io"
	"log"
	"net/http"
	"time"
)

const (
	templatePath = "templates/*"
	sfspca       = "https://www.sfspca.org/wp-json/sfspca/v1/filtered-posts/get-adoptions?current-term[id]=94&current-term[taxonomy]=species&ignored-terms[sfspca-adoption-site][]=74&ignored-terms[sfspca-adoption-site][]=128&ignored-terms[sfspca-adoption-site][]=485&ignored-terms[sfspca-adoption-gender][]=354&order=ASC&orderby=date&page=1&per_page=100"
)

type ServerContext struct {
	mg         *mailgun.MailgunImpl
	recipients []string
}

var doggos SFSPCA_Response

type Doggo struct {
	Title string
	Tags  struct {
		Gender         string
		WeightCategory string `json:"weight-category"`
		Species        string
		Breed          string
		Color          string
		Location       string
		Site           string
	}
	Permalink string
	Thumb     []string
	Age       string
}

type SFSPCA_Response struct {
	Items     []Doggo
	Total     int16
	Displayed int16
}

func startWebServer(sc ServerContext, resp SFSPCA_Response) {
	r := gin.Default()
	r.LoadHTMLGlob(templatePath)
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "doggos.html", gin.H{
			"title":  "SFSPCA Doggos",
			"doggos": resp,
		})
	})
	r.GET("/emails/send", func(c *gin.Context) {
		for _, r := range sc.recipients {
			sendMail(sc.mg, r, doggos)
		}
		c.String(http.StatusOK, "Sent!")
	})
	r.Run()
}

func getDoggos() SFSPCA_Response {
	resp, err := http.Get(sfspca)
	if err != nil {
		log.Panic(err)
	}

	dog_bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Panic(err)
	}

	var response_object SFSPCA_Response
	err = json.Unmarshal(dog_bytes, &response_object)
	if err != nil {
		log.Panic(err)
	}

	return response_object
}

func main() {
	viper.SetConfigFile("config.yaml")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		log.Panicf("Fatal error config file: %w \n", err)
	}

	emailDomain := viper.GetString("mailgun.domain")
	privateAPIKey := viper.GetString("mailgun.private_key")
	recipients := viper.GetStringSlice("emails")
	log.Printf("emails subscribed: %s", recipients)

	mg := mailgun.NewMailgun(emailDomain, privateAPIKey)

	sc := ServerContext{
		mg,
		recipients,
	}

	s := gocron.NewScheduler(time.UTC)

	doggos = getDoggos()

	s.Every(1).Day().At("14:00").Do(func() {
		doggos = getDoggos()
		for _, recipient := range recipients {
			sendMail(mg, recipient, doggos)
		}
	})

	startWebServer(sc, doggos)
}
