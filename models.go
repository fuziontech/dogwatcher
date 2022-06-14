package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type DoggoStatus struct {
	New       []Doggo
	Adopted   []Doggo
	Available []Doggo
}

type JSONDoggo struct {
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

type Doggo struct {
	ID             uint `gorm:"primarykey"`
	Title          string
	Gender         string
	WeightCategory string
	Species        string
	Breed          string
	Color          string
	Location       string
	Site           string
	Permalink      string
	JSONThumbURLs  datatypes.JSON
	JSONThumbs     datatypes.JSON
	Age            string
	AdoptedAt      sql.NullTime
	LastSeen       time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`
}

type SFSPCAResponse struct {
	Items     []JSONDoggo
	Total     int16
	Displayed int16
}

func autoMigrate(ctx ServerContext) error {
	err := ctx.gdb.AutoMigrate(&Doggo{}, &Email{})
	if err != nil {
		log.Panicf("failed to migrate %s", err)
	}
	return err
}

func (jsonDoggo JSONDoggo) toDoggoModel() Doggo {
	ID64, err := strconv.ParseUint(strings.Split(jsonDoggo.Permalink, "/")[5], 10, 32)
	if err != nil {
		log.Panicf("Unable to get id from permalink for doggo %s", err)
	}
	ID := uint(ID64)

	doggo := Doggo{}

	thumbs, err := json.Marshal(jsonDoggo.Thumb)
	if err != nil {
		log.Panicf("could not serialize the thumbs %s", err)
	}

	doggo.ID = ID
	doggo.Title = jsonDoggo.Title
	doggo.Gender = jsonDoggo.Tags.Gender
	doggo.WeightCategory = jsonDoggo.Tags.WeightCategory
	doggo.Species = jsonDoggo.Tags.Species
	doggo.Breed = jsonDoggo.Tags.Breed
	doggo.Color = jsonDoggo.Tags.Color
	doggo.Location = jsonDoggo.Tags.Location
	doggo.Site = jsonDoggo.Tags.Site
	doggo.Permalink = jsonDoggo.Permalink
	doggo.JSONThumbURLs = thumbs
	doggo.Age = jsonDoggo.Age
	doggo.LastSeen = time.Now()

	return doggo
}

func (doggo Doggo) ThumbURLs() []string {
	var thumbs []string
	err := json.Unmarshal(doggo.JSONThumbURLs, &thumbs)
	if err != nil {
		log.Panicf("cannot get thumb urls from json urls %s", err)
	}
	return thumbs
}

func (doggo Doggo) ThumbBytes() [][]byte {
	var thumbs [][]byte
	err := json.Unmarshal(doggo.JSONThumbs, &thumbs)
	if err != nil {
		log.Panicf("cannot get thumb urls from json bytes %s", err)
	}
	return thumbs
}

func (doggo Doggo) fillThumbs() Doggo {
	var thumbs [][]byte
	var thumbURLs []string
	err := json.Unmarshal(doggo.JSONThumbURLs, &thumbURLs)
	if err != nil {
		log.Panicf("unable to unmarshal thumb urls %s with error %s", doggo.JSONThumbURLs, err)
	}

	for _, thumbURL := range thumbURLs {
		resp, err := http.Get(string(thumbURL))
		if err != nil {
			log.Panicf("couldn't get the thumb at %s: %s", thumbURL, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			log.Panicf("trying to get the thumb and recieved status != 200: %d", resp.StatusCode)
		}
		thumb, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Panicf("couldn't read the thumb returned %s", err)
		}
		thumbs = append(thumbs, thumb)
	}
	jthumbs, err := json.Marshal(thumbs)
	doggo.JSONThumbs = jthumbs
	if err != nil {
		log.Panicf("couldn't marshal the thumbs into json %s", err)
	}
	return doggo
}

func saveDoggo(ctx ServerContext, doggo Doggo) error {
	doggo = doggo.fillThumbs()
	return ctx.gdb.Create(&doggo).Error
}

func saveDoggos(ctx ServerContext, doggos []Doggo) error {
	for _, doggo := range doggos {
		err := saveDoggo(ctx, doggo)
		if err != nil {
			log.Printf("could not save doggo %s", err)
			return err
		}
	}
	return nil
}

func findAdoptedDoggos(ctx ServerContext, response SFSPCAResponse) ([]Doggo, error) {
	adoptedDoggos := []Doggo{}
	var ids []uint
	for _, d := range response.Items {
		doggo := d.toDoggoModel()
		ids = append(ids, doggo.ID)
	}
	err := ctx.gdb.Not(ids).Where("adopted_at is Null").Find(&adoptedDoggos).Error
	if err != nil {
		return []Doggo{}, err
	}
	err = ctx.gdb.Model(Doggo{}).Not(ids).Where("adopted_at is Null").Update("adopted_at", time.Now()).Error
	if err != nil {
		return []Doggo{}, err
	}
	return adoptedDoggos, nil
}

func findNewlyListedDoggos(ctx ServerContext, response SFSPCAResponse) ([]Doggo, error) {
	var newlyListedDoggos []Doggo
	for _, d := range response.Items {
		doggo := d.toDoggoModel()
		dbDoggo := Doggo{}
		err := ctx.gdb.First(&dbDoggo, doggo.ID).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			newlyListedDoggos = append(newlyListedDoggos, doggo)
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return newlyListedDoggos, err
		}
	}
	return newlyListedDoggos, nil
}

func fetchAvailableDoggos(ctx ServerContext) ([]Doggo, error) {
	var doggos []Doggo
	err := ctx.gdb.Where("adopted_at is null").Omit("json_thumbs").Find(&doggos).Error
	if err != nil {
		return doggos, err
	}
	return doggos, err
}

func fetchAdoptedDoggos(ctx ServerContext) ([]Doggo, error) {
	yesterday := time.Now().Add(-time.Hour * 24)
	var doggos []Doggo
	err := ctx.gdb.Where("adopted_at >= ?", yesterday).Find(&doggos).Error
	if err != nil {
		return doggos, err
	}
	return doggos, err
}

func fetchNewDoggos(ctx ServerContext) ([]Doggo, error) {
	yesterday := time.Now().Add(-time.Hour * 24)
	var doggos []Doggo
	err := ctx.gdb.Where("created_at >= ?", yesterday).Omit("json_thumbs").Find(&doggos).Error
	if err != nil {
		return doggos, err
	}
	return doggos, err
}

func fetchDBDoggos(ctx ServerContext) (DoggoStatus, error) {
	available, err := fetchAvailableDoggos(ctx)
	if err != nil {
		return DoggoStatus{}, err
	}
	newDoggos, err := fetchNewDoggos(ctx)
	if err != nil {
		return DoggoStatus{}, err
	}
	adoptedDoggos, err := fetchAdoptedDoggos(ctx)
	if err != nil {
		return DoggoStatus{}, err
	}
	doggos := DoggoStatus{
		Available: available,
		New:       newDoggos,
		Adopted:   adoptedDoggos,
	}
	return doggos, nil
}
