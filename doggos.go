package main

import (
	"fmt"
	"log"
)

func updateDoggos(ctx ServerContext, resp SFSPCAResponse) (DoggoStatus, error) {
	// Load DB with Doggos and detect newly listed Doggos
	newDoggos, err := findNewlyListedDoggos(ctx, resp)
	if err != nil {
		log.Panicf("could not determine which doggos are newly listed %s", err)
		return DoggoStatus{}, err
	}

	if len(newDoggos) > 0 {
		fmt.Println("NEWLY LISTED DOGGOS:")
		for _, d := range newDoggos {
			fmt.Printf("%s\n", d.Title)
		}

		err = saveDoggos(ctx, newDoggos)
		if err != nil {
			log.Panicf("could not save doggos %s", err)
			return DoggoStatus{}, err
		}
	}

	// Detect adopted doggos
	adoptedDoggos, err := findAdoptedDoggos(ctx, resp)
	if err != nil {
		log.Panicf("could not determine which dogs have been adopted %s", err)
		return DoggoStatus{}, err
	}

	if len(adoptedDoggos) > 0 {
		fmt.Println("DOGGOS FOUND A HOME!")
		for _, d := range adoptedDoggos {
			fmt.Printf("%s\n", d.Title)
		}
	}

	ds := DoggoStatus{
		Adopted: adoptedDoggos,
		New:     newDoggos,
	}

	ds.Available, err = fetchAvailableDoggos(ctx)
	if err != nil {
		return DoggoStatus{}, err
	}
	return ds, nil
}

func fetchAndUpdateDoggos(ctx ServerContext) error {
	resp := fetchDoggos()
	_, err := updateDoggos(ctx, resp)
	if err != nil {
		log.Panicf("could not update doggos %s", err)
		return err
	}
	return nil
}
