package main

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-co-op/gocron"
	"github.com/mailgun/mailgun-go/v4"
	"github.com/spf13/viper"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
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
	gdb        *gorm.DB
	mg         *mailgun.MailgunImpl
	recipients []string
}

var doggos SFSPCAResponse

func startWebServer(sc ServerContext) {
	r := gin.Default()
	r.LoadHTMLGlob(templatePath)
	r.Static("/static", "./static")
	r.StaticFile("/favicon.ico", "./static/favicon.ico")
	r.GET("/", func(c *gin.Context) {
		doggos, err := fetchDBDoggos(sc)
		if err != nil {
			c.Error(err)
		}

		c.HTML(http.StatusOK, "doggos.html", gin.H{
			"title":      "SFSPCA Doggos",
			"doggos":     doggos,
			"doggoCount": len(doggos),
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

func fetchDoggos() SFSPCAResponse {
	resp, err := http.Get(sfspca)
	if err != nil {
		log.Panic(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Panicf("didn't get an OK response from SFSPCA: %s", resp.StatusCode)
	}

	dog_bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Panic(err)
	}

	var response_object SFSPCAResponse
	err = json.Unmarshal(dog_bytes, &response_object)
	if err != nil {
		log.Panic(err)
	}

	return response_object
}

func main() {
	// Load configs
	viper.SetConfigFile("config.yaml")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		log.Panicf("Fatal error config file: %w \n", err)
	}

	postgresURL := viper.GetString("postgres.url")
	emailDomain := viper.GetString("mailgun.domain")
	privateAPIKey := viper.GetString("mailgun.private_key")
	recipients := viper.GetStringSlice("emails")
	log.Printf("emails subscribed: %s", recipients)

	// Configure Postgres
	db, err := gorm.Open(postgres.Open(postgresURL), &gorm.Config{})
	if err != nil {
		log.Panicf("failed to connect to postgres with error %s", err)
	}

	err = db.AutoMigrate(&Doggo{})
	if err != nil {
		log.Panicf("failed to migrate %s", err)
	}

	// Configure mailgun
	mg := mailgun.NewMailgun(emailDomain, privateAPIKey)

	// Configure server context
	sc := ServerContext{
		db,
		mg,
		recipients,
	}

	// Preload doggos
	doggos = fetchDoggos()

	// configure CRON
	s := gocron.NewScheduler(time.UTC)
	s.Every(1).Day().At("14:00").Do(func() {
		doggos = fetchDoggos()
		for _, recipient := range recipients {
			sendMail(mg, recipient, doggos)
		}
	})
	s.StartAsync()

	// Load DB with Doggos and detect newly listed Doggos
	newDoggos, err := findNewlyListedDoggos(sc, doggos)
	if err != nil {
		log.Panicf("could not determine which doggos are newly listed %s", err)
	}

	if len(newDoggos) > 0 {
		fmt.Println("NEWLY LISTED DOGGOS:")
		for _, d := range newDoggos {
			fmt.Printf("%s\n", d.Title)
		}

		err = saveDoggos(sc, newDoggos)
		if err != nil {
			log.Panicf("could not save doggos %s", err)
		}
	}

	// Detect adopted doggos
	adoptedDoggos, err := findAdoptedDoggos(sc, doggos)
	if err != nil {
		log.Panicf("could not determine which dogs have been adopted %s", err)
	}

	if len(adoptedDoggos) > 0 {
		fmt.Println("DOGGOS FOUND A HOME!")
		for _, d := range adoptedDoggos {
			fmt.Printf("%s\n", d.Title)
		}
	}

	startWebServer(sc)
}
