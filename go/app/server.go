package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"io"
	"crypto/sha256"
	"strconv"
)

type Server struct {
	// Port is the port number to listen on.
	Port string
	// ImageDirPath is the path to the directory storing images.
	ImageDirPath string
}

// Run is a method to start the server.
// This method returns 0 if the server started successfully, and 1 otherwise.
func (s Server) Run() int {
	// set up logger
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	slog.SetDefault(logger)
	// STEP 4-6: set the log level to DEBUG
	slog.SetLogLoggerLevel(slog.LevelInfo)

	// set up CORS settings
	frontURL, found := os.LookupEnv("FRONT_URL")
	if !found {
		frontURL = "http://localhost:3000"
	}

	// STEP 5-1: set up the database connection
	db, err := InitDB("db/mercari.sqlite3")
	if err != nil {
		slog.Error("failed to initialize database", "error", err)
		return 1
	}

	// set up handlers
	itemRepo := NewItemRepository(db)
	h := &Handlers{imgDirPath: s.ImageDirPath, itemRepo: itemRepo}

	// set up routes
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", h.Hello)
	mux.HandleFunc("POST /items", h.AddItem)
	mux.HandleFunc("GET /items", h.GetItems)//add in 4-3
	mux.HandleFunc("GET /images/{filename}", h.GetImage)
	mux.HandleFunc("GET /items/{item_id}", h.GetItem)
	mux.HandleFunc("GET /search", h.Search)

	// start the server
	slog.Info("http server started on", "port", s.Port)
	err = http.ListenAndServe(":"+s.Port, simpleCORSMiddleware(simpleLoggerMiddleware(mux), frontURL, []string{"GET", "HEAD", "POST", "OPTIONS"}))
	if err != nil {
		slog.Error("failed to start server: ", "error", err)
		return 1
	}

	return 0
}

type Handlers struct {
	// imgDirPath is the path to the directory storing images.
	imgDirPath string
	itemRepo   ItemRepository
}

type HelloResponse struct {
	Message string `json:"message"`
}

// Hello is a handler to return a Hello, world! message for GET / .
func (s *Handlers) Hello(w http.ResponseWriter, r *http.Request) {
	resp := HelloResponse{Message: "Hello, world!"}
	err := json.NewEncoder(w).Encode(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type AddItemRequest struct {
	Name string `json:"name"`
	Category string `json:"category"`// STEP 4-2: add a category field
	Image []byte `json:"image_name"` // STEP 4-4: add an image field
}

type AddItemResponse struct {
	Message string `json:"message"`
}

// parseAddItemRequest parses and validates the request to add an item.
func parseAddItemRequest(r *http.Request) (*AddItemRequest, []byte, error) {
	req := &AddItemRequest{
		Name: r.FormValue("name"),
		Category: r.FormValue("category"),// STEP 4-2: add a category field
	}

	// STEP 4-4: add an image field
	err := r.ParseMultipartForm(10 << 20)
        if err != nil {
        fmt.Println("Failed to parse form data:", err)
        return nil, nil, fmt.Errorf("failed to parse form data: %w", err)
        }

        file, _, err := r.FormFile("image")
        if err != nil {
                return nil, nil, errors.New("image file is required")
    }
        defer file.Close()
	 
	// validate the request
	if req.Name == "" {
		return nil, nil, errors.New("name is required")
	}

	// STEP 4-2: validate the category field
	if req.Category == ""{
		return nil, nil, errors.New("category is required")
	}

	// STEP 4-4: validate the image field
	imageData, err:= io.ReadAll(file)
	if err != nil {
		return nil, nil, errors.New("imagename is required")
	}

	return req, imageData, nil
}

// AddItem is a handler to add a new item for POST /items .
func (s *Handlers) AddItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	req, imageData, err := parseAddItemRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// STEP 4-4: uncomment on adding an implementation to store an image
	filePath, err := s.storeImage(imageData)
	if err != nil {
		slog.Error("failed to store image: ", "error", err)
	 	http.Error(w, err.Error(), http.StatusInternalServerError)
	 	return
	}

	item := &Item{
		Name: req.Name,
		Category: req.Category,// STEP 4-2: add a category field
		Image : filePath,// STEP 4-4: add an image field
	}
	message := fmt.Sprintf("item received: %s", item.Name)
	slog.Info(message)

	// STEP 4-2: add an implementation to store an image
	err = s.itemRepo.Insert(ctx, item)
	if err != nil {
		slog.Error("failed to store item: ", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := AddItemResponse{Message: message}
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// GetItems is a handler to return resistered items
func (s *Handlers) GetItems(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // get the data
    items, err := s.itemRepo.GetAll(ctx)
    if err != nil {
        http.Error(w, "filed to retrieve items", http.StatusInternalServerError)
        return
    }

    // return JSON response
    resp := struct {
        Items []Item `json:"items"`
    }{Items: items}

    json.NewEncoder(w).Encode(resp)
}


// storeImage stores an image and returns the file path and an error if any.
// this method calculates the hash sum of the image as a file name to avoid the duplication of a same file
// and stores it in the image directory.
func (s *Handlers) storeImage(image []byte) (filePath string, err error) {
	// STEP 4-4: add an implementation to store an image

	// TODO:
	// - calc hash sum
	hash := sha256.Sum256(image)
	fileName := fmt.Sprintf("%x.jpg",hash)

	// - build image file path
	filePath = filepath.Join(s.imgDirPath, fileName)

	// - check if the image already exists
	if _, err = os.Stat(filePath); err == nil{
		return fileName, nil
	}

	// - store image
	fmt.Println("Savin new image:",fileName)
	err = os.WriteFile(filePath, image, 0644)
	if err != nil{
		return "", fmt.Errorf("failed to save image: %w", err)
	}

	// - return the image file path

	return fileName, nil
}

type GetImageRequest struct {
	FileName string // path value
}

// parseGetImageRequest parses and validates the request to get an image.
func parseGetImageRequest(r *http.Request) (*GetImageRequest, error) {
	req := &GetImageRequest{
		FileName: r.PathValue("filename"), // from path parameter
	}

	// validate the request
	if req.FileName == "" {
		return nil, errors.New("filename is required")
	}

	return req, nil
}

// GetImage is a handler to return an image for GET /images/{filename} .
// If the specified image is not found, it returns the default image.
func (s *Handlers) GetImage(w http.ResponseWriter, r *http.Request) {
	req, err := parseGetImageRequest(r)
	if err != nil {
		slog.Warn("failed to parse get image request: ", "error", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	imgPath, err := s.buildImagePath(req.FileName)
	if err != nil {
		if !errors.Is(err, errImageNotFound) {
			slog.Warn("failed to build image path: ", "error", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// when the image is not found, it returns the default image without an error.
		slog.Debug("image not found", "filename", imgPath)
		imgPath = filepath.Join(s.imgDirPath, "default.jpg")
	}

	slog.Info("returned image", "path", imgPath)
	http.ServeFile(w, r, imgPath)
}

// buildImagePath builds the image path and validates it.
func (s *Handlers) buildImagePath(imageFileName string) (string, error) {
	imgPath := filepath.Join(s.imgDirPath, filepath.Clean(imageFileName))

	// to prevent directory traversal attacks
	rel, err := filepath.Rel(s.imgDirPath, imgPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("invalid image path: %s", imgPath)
	}

	// validate the image suffix
	if !strings.HasSuffix(imgPath, ".jpg") && !strings.HasSuffix(imgPath, ".jpeg") {
		return "", fmt.Errorf("image path does not end with .jpg or .jpeg: %s", imgPath)
	}

	// check if the image exists
	_, err = os.Stat(imgPath)
	if err != nil {
		return imgPath, errImageNotFound
	}

	return imgPath, nil
}

//// parseGetItemRequest parses and validates the request to get an item information.
func parseGetItemRequest(r *http.Request) (int, error) {
	itemIDStr := r.PathValue("item_id")
	itemID, err := strconv.Atoi(itemIDStr)
	if err != nil || itemID < 1 {
		return 0, errors.New("invalid item ID")
	}
	return itemID, nil
}

// GetItem is a handler to return an item information for GET /images/{item_id} .
func (s *Handlers) GetItem(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	itemID, err := parseGetItemRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	item, err := s.itemRepo.GetByID(ctx,itemID)
	if err != nil {
		http.Error(w, "item not found", http.StatusNotFound)
		return
	}

	resp, err := json.Marshal(item)
	if err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func (s *Handlers) Search(w http.ResponseWriter, r *http.Request){
	ctx := r.Context()

	keyword := r.URL.Query().Get("keyword")
	if keyword == ""{
		http.Error(w, "keyword is required", http.StatusBadRequest)
		return 
	}

	items, err := s.itemRepo.SearchByKeyword(ctx, keyword)
	if err != nil{
		http.Error(w, "failed to search items", http.StatusInternalServerError)
		return
	}

	resp := struct{
		Items []Item `json:"items"`
	}{Items: items}

	json.NewEncoder(w).Encode(resp)
}


