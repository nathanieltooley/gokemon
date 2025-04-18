package main

import (
	"encoding/json"
	"io"
	"net/http"
	"os"

	"github.com/nathanieltooley/gokemon/client/game"
)

type Response struct {
	Items []game.NamedApiResource
}

func main() {
	itemResponse, err := http.Get("https://pokeapi.co/api/v2/item-attribute/7/")
	if err != nil {
		panic(err)
	}

	parsedResponse := new(Response)
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

	itemFileName := "./data/items.json"
	os.Remove(itemFileName)

	itemsFile, err := os.Create(itemFileName)
	defer itemsFile.Close()
	if err != nil {
		panic(err)
	}

	itemsBytes, err := json.Marshal(items)
	if err != nil {
		panic(err)
	}

	if _, err := itemsFile.Write(itemsBytes); err != nil {
		panic(err)
	}
}
