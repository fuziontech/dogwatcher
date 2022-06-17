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
	isProduction bool
}

func fetchDoggos() SFSPCAResponse {
	resp, err := http.Get(sfspca)
	if err != nil {
		log.Panic(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Panicf("didn't get an OK response from SFSPCA: %d", resp.StatusCode)
	}

	dogBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Panic(err)
	}

	var responseObject SFSPCAResponse
	err = json.Unmarshal(dogBytes, &responseObject)
	if err != nil {
		log.Panicf("Cannot unmarshal string %s because of error %s", dogBytes, err)
	}

	return responseObject
}

func main() {
	// Load configs
	viper.SetConfigFile("config.yaml")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		log.Panicf("Fatal error config file: %s \n", err)
	}

	isProd := viper.GetBool("production")

	postgresURL := viper.GetString("postgres.url")
	webDomain := viper.GetString("web.domain")
	emailDomain := viper.GetString("mailgun.domain")
	privateAPIKey := viper.GetString("mailgun.private_key")

	// Configure Postgres
	db, err := gorm.Open(postgres.Open(postgresURL), &gorm.Config{})
	if err != nil {
		log.Panicf("failed to connect to postgres with error %s", err)
	}

	// Configure mailgun
	mg := mailgun.NewMailgun(emailDomain, privateAPIKey)

	// Configure server context
	ctx := ServerContext{
		db,
		mg,
		isProd,
	}

	err = autoMigrate(ctx)
	if err != nil {
		log.Panicf("Cannot automigrate %s\n", err)
	}

	// configure CRON
	s := gocron.NewScheduler(time.UTC)
	s.Every(1).Day().At("14:00").Do(func() {
		resp := fetchDoggos()
		doggos, err := updateDoggos(ctx, resp)
		if err != nil {
			log.Panicf("cannot update doggos %s", err)
		}
		recipients, err := getAllEmails(ctx)
		if err != nil {
			log.Panicf("cannot get emails %s", err)
		}
		doggos, err = fetchDBDoggos(ctx)
		if err != nil {
			log.Panicf("cannot get db doggos %s", err)
		}
		for _, recipient := range recipients {
			sendMail(mg, recipient.Email, doggos)
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
