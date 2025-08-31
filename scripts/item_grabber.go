package main

import (
	"encoding/json"
	"io"
	"net/http"
	"os"

	"github.com/nathanieltooley/gokemon/golurk"
)

type ItemResponse struct {
	Items []golurk.NamedApiResource
}

func itemMain(itemFileName string) {
	itemResponse, err := http.Get("https://pokeapi.co/api/v2/item-attribute/7/")
	if err != nil {
		panic(err)
	}

	parsedResponse := new(ItemResponse)
	responseBytes, err := io.ReadAll(itemResponse.Body)
	if err != nil {
		panic(err)
	}

	if err := json.Unmarshal(responseBytes, parsedResponse); err != nil {
		panic(err)
	}

	items := make([]string, len(parsedResponse.Items))
	for i, item := range parsedResponse.Items {
		items[i] = item.Name
	}

	os.Remove(itemFileName)

	itemsFile, err := os.Create(itemFileName)
	if err != nil {
		panic(err)
	}
	defer itemsFile.Close()

	itemsBytes, err := json.Marshal(items)
	if err != nil {
		panic(err)
	}

	if _, err := itemsFile.Write(itemsBytes); err != nil {
		panic(err)
	}
}
