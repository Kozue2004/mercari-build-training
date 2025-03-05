package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	// STEP 5-1: uncomment this line
	// _ "github.com/mattn/go-sqlite3"
)

var errImageNotFound = errors.New("image not found")

type Item struct {
	ID       int    `db:"id" json:"-"`
	Name     string `db:"name" json:"name"`
	Category string `db:"category" json:"category"`
	Image    string `db:"image_name" json:"image_name"`
}

// Please run `go generate ./...` to generate the mock implementation
// ItemRepository is an interface to manage items.
//
//go:generate go run go.uber.org/mock/mockgen -source=$GOFILE -package=${GOPACKAGE} -destination=./mock_$GOFILE
type ItemRepository interface {
	Insert(ctx context.Context, item *Item) error
	GetAll(ctx context.Context) ([]Item, error)
}

// itemRepository is an implementation of ItemRepository
type itemRepository struct {
	// fileName is the path to the JSON file storing items.
	fileName string
}

// NewItemRepository creates a new itemRepository.
func NewItemRepository() ItemRepository {
	return &itemRepository{fileName: "items.json"}
}

// Insert inserts an item into the repository.
func (i *itemRepository) Insert(ctx context.Context, item *Item) error {
	// STEP 4-1: add an implementation to store an item
	file, err := os.ReadFile(i.fileName)
	if err != nil {
		if os.IsNotExist(err) {
			file = []byte(`{"items": []}`)
		} else {
			return fmt.Errorf("failed to read file: %w", err)
		}
	}

	if len(file) == 0 {
		file = []byte(`{"items": []}`)
	}

	var data struct {
		Items []Item `json:"items"`
	}
	err = json.Unmarshal(file, &data)
	if err != nil {
		return fmt.Errorf("faild to parse JSON:%w", err)
	}

	data.Items = append(data.Items, *item)

	updatedData, err := json.MarshalIndent(data, "", " ")
	if err != nil {
		return fmt.Errorf("failed to encode JSON:%w", err)
	}

	err = os.WriteFile(i.fileName, updatedData, 0644)
	if err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}

	return nil
}

func (i *itemRepository) GetAll(ctx context.Context) ([]Item, error) {
	file, err := os.ReadFile(i.fileName)
	if err != nil {
		if os.IsNotExist(err) {
			return []Item{}, nil
		}
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var data struct {
		Items []Item `json:"items"`
	}
	err = json.Unmarshal(file, &data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return data.Items, nil
}

// StoreImage stores an image and returns an error if any.
// This package doesn't have a related interface for simplicity.
func StoreImage(fileName string, image []byte) error {
	// STEP 4-4: add an implementation to store an image
	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	_, err = file.Write(image)
	if err != nil {
		return fmt.Errorf("failed to write image data: %w", err)
	}

	return nil
}
