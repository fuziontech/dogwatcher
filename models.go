package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"log"
	"strconv"
	"strings"
	"time"
)

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
	Thumb          datatypes.JSON
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
	doggo.Thumb = thumbs
	doggo.Age = jsonDoggo.Age
	doggo.LastSeen = time.Now()

	return doggo
}

func saveDoggo(sc ServerContext, doggo Doggo) error {
	return sc.gdb.Create(&doggo).Error
}

func saveDoggos(sc ServerContext, doggos []Doggo) error {
	for _, doggo := range doggos {
		err := saveDoggo(sc, doggo)
		if err != nil {
			log.Panicf("could not save doggo %s", err)
			return err
		}
	}
	return nil
}

func findAdoptedDoggos(sc ServerContext, response SFSPCAResponse) ([]Doggo, error) {
	adoptedDoggos := []Doggo{}
	var ids []uint
	for _, d := range response.Items {
		doggo := d.toDoggoModel()
		ids = append(ids, doggo.ID)
	}
	err := sc.gdb.Not(ids).Where("adopted_at = ?", nil).Find(&adoptedDoggos).Error
	if err != nil {
		return []Doggo{}, err
	}
	return adoptedDoggos, err
}

func findNewlyListedDoggos(sc ServerContext, response SFSPCAResponse) ([]Doggo, error) {
	var newlyListedDoggos []Doggo
	for _, d := range response.Items {
		doggo := d.toDoggoModel()
		dbDoggo := Doggo{}
		err := sc.gdb.First(&dbDoggo, doggo.ID).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			newlyListedDoggos = append(newlyListedDoggos, doggo)
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return newlyListedDoggos, err
		}
	}
	return newlyListedDoggos, nil
}
