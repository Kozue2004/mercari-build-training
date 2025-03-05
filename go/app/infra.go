package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"database/sql"
	// STEP 5-1: uncomment this line
	_ "github.com/mattn/go-sqlite3"
)

var errImageNotFound = errors.New("image not found")
var db *sql.DB

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
	GetByID(ctx context.Context, itemID int) (*Item, error)
	SearchByKeyword(ctx context.Context, keyword string) ([]Item, error)
}

// itemRepository is an implementation of ItemRepository
type itemRepository struct {
	db *sql.DB
}

// NewItemRepository creates a new itemRepository.
func NewItemRepository(database *sql.DB) ItemRepository {
	return &itemRepository{db: database}
}

// Insert inserts an item into the repository.
func (i *itemRepository) Insert(ctx context.Context, item *Item) error {
	// STEP 5-1: add an implementation to store an item
	_, err := i.db.ExecContext(ctx, "INSERT INTO items (name, category, image_name) VALUES (?, ?, ?)", item.Name, item.Category, item.Image)
	if err != nil{
		return fmt.Errorf("failed to insert item :%w",err)
	}
	return nil
}

func (i *itemRepository) GetAll(ctx context.Context) ([]Item, error) {
	rows, err := i.db.QueryContext(ctx, "SELECT id, name, category, image_name FROM items")
	if err != nil{
		return nil, fmt.Errorf("failed to get items: %w", err)
	}
	defer rows.Close()

	var items []Item
	for rows.Next(){
		var item Item
		if err := rows.Scan(&item.ID, &item.Name, &item.Category, &item.Image); err != nil{
			return nil, fmt.Errorf("failed to scan item: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil{
		return nil, fmt.Errorf("rows nteraction error: %w", err)
	}

	return items, nil
}

func (i *itemRepository) GetByID(ctx context.Context, itemID int) (*Item, error) {
    var item Item
    err := i.db.QueryRowContext(ctx, "SELECT id, name, category, image FROM items WHERE id = ?", itemID).
        Scan(&item.ID, &item.Name, &item.Category, &item.Image)

    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return nil, fmt.Errorf("item not found")
        }
        return nil, fmt.Errorf("failed to query item: %w", err)
    }
    return &item, nil
}

func (i *itemRepository) SearchByKeyword(ctx context.Context, keyword string) ([]Item, error){
	rows, err := i.db.QueryContext(ctx, "SELECT id, name, category, image_name FROM items WHERE name LIKE ?", "%"+keyword+"%")
	if err != nil{
		return nil, fmt.Errorf("failed to search items: %w", err)
	}
	defer rows.Close()

	var items []Item
	for rows.Next(){
		var item Item
		if err := rows.Scan(&item.ID, &item.Name, &item.Category, &item.Image); err != nil{
			return nil, fmt.Errorf("failed to scan item: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil{
		return nil, fmt.Errorf("rows interaction error: %w",err)
	}

	return items, nil
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

func InitDB(dbPath string) (*sql.DB, error){
	database, err := sql.Open("sqlite3",dbPath)
	if err != nil{
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	createTableQuery := `
	CREATE TABLE IF NOT EXISTS items(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		category TEXT NOT NULL,
		image_name TEXT NOT NULL
	);`
	_, err = database.Exec(createTableQuery)
	if err != nil{
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	return database, nil
}




