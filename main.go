package main

import (
	"encoding/json"
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
	gdb          *gorm.DB
	mg           *mailgun.MailgunImpl
	recipients   []string
	isProduction bool
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

	isProd := viper.GetBool("production")

	postgresURL := viper.GetString("postgres.url")
	webDomain := viper.GetString("web.domain")
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
	ctx := ServerContext{
		db,
		mg,
		recipients,
		isProd,
	}

	// configure CRON
	s := gocron.NewScheduler(time.UTC)
	s.Every(1).Day().At("14:00").Do(func() {
		resp := fetchDoggos()
		doggos, err := updateDoggos(ctx, resp)
		if err != nil {
			log.Panicf("cannot update doggos %s", err)
		}
		for _, recipient := range recipients {
			sendMail(mg, recipient, doggos)
		}
	})
	s.Every(1).Hour().Do(func() {
		fetchAndUpdateDoggos(ctx)
		if err != nil {
			log.Panicf("cannot update doggos %s", err)
		}
	})
	s.StartAsync()

	fetchAndUpdateDoggos(ctx)
	startWebServer(ctx, webDomain)
}
