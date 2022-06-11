package main

import (
	"encoding/json"
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

	return doggo
}

func saveDoggo(sc ServerContext, jsonDoggo JSONDoggo) error {
	doggo := jsonDoggo.toDoggoModel()
	return sc.gdb.Create(&doggo).Error
}

func saveDoggos(sc ServerContext, sfspca SFSPCAResponse) error {
	for _, doggo := range sfspca.Items {
		err := saveDoggo(sc, doggo)
		if err != nil {
			log.Panicf("could not save doggo %s", err)
			return err
		}
	}
	return nil
}
