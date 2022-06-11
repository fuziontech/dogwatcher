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

const templatePath = "templates/*"

const sfspca = "https://www.sfspca.org/wp-json/sfspca/v1/filtered-posts/get-adoptions?current-term%5Bid%5D=94&current-term%5Btaxonomy%5D=species&ignored-terms%5Bsfspca-adoption-site%5D%5B%5D=74&ignored-terms%5Bsfspca-adoption-site%5D%5B%5D=128&ignored-terms%5Bsfspca-adoption-site%5D%5B%5D=485&ignored-terms%5Bsfspca-adoption-gender%5D%5B%5D=354&order=ASC&orderby=date"

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

func startWebServer(resp SFSPCA_Response) {
	r := gin.Default()
	r.LoadHTMLGlob(templatePath)
	r.GET("/index", func(c *gin.Context) {
		c.HTML(http.StatusOK, "doggos.html", gin.H{
			"title":  "SFSPCA Doggos",
			"doggos": resp,
		})
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

	mg := mailgun.NewMailgun(emailDomain, privateAPIKey)

	s := gocron.NewScheduler(time.Local)

	doggos = getDoggos()

	s.Every(1).Day().At("7:00").Do(func() {
		doggos = getDoggos()
		sendMail(mg, doggos)
	})

	startWebServer(doggos)
}
