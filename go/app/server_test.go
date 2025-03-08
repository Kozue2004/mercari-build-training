package app

import (
	"bytes"          //add in STEP6-1
	"database/sql"   //add in STEP6-4
	"encoding/json"  //add in STEP6-2
	"fmt"            //add in STEP6-3
	"io"             //add in STEP6-1
	"mime/multipart" //add in STEP6-1
	"net/http"
	"net/http/httptest"
	"os"            //add in STEP6-1
	"path/filepath" //add in STEP6-1
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	_ "github.com/mattn/go-sqlite3" //add in STEP6-4
	"go.uber.org/mock/gomock"
)

func TestParseAddItemRequest(t *testing.T) {
	t.Parallel()

	type wants struct {
		req *AddItemRequest
		err bool
	}

	// STEP 6-1: define test cases
	cases := map[string]struct {
		args     map[string]string
		filePath string
		wants
	}{
		"ok: valid request": {
			args: map[string]string{
				"name":     "TestName",     // fill here
				"category": "TestCategory", // fill here
			},
			filePath: "test.png", // imagefile
			wants: wants{
				req: &AddItemRequest{
					Name:     "TestName",     // fill here
					Category: "TestCategory", // fill here
					Image:    nil,            //check the filename
				},
				err: false,
			},
		},
		"ng: empty request": {
			args:     map[string]string{},
			filePath: "",
			wants: wants{
				req: nil,
				err: true,
			},
		},
	}

	for name, tt := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// prepare request body as multipart/form-data
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)
			writer.WriteField("name", "TestName")
			writer.WriteField("category", "TestCategory")

			if tt.filePath != "" {
				file, err := os.Open(tt.filePath)
				if err != nil {
					t.Fatalf("failed to open image file: %v", err)
				}
				defer file.Close()
				part, err := writer.CreateFormFile("image", filepath.Base(tt.filePath))
				if err != nil {
					t.Fatalf("failed to create form file for image: %v", err)
				}
				_, err = io.Copy(part, file)
				if err != nil {
					t.Fatalf("failed to copy image content: %v", err)
				}
			} //add image to request body

			writer.Close()

			// prepare HTTP request
			req, err := http.NewRequest("POST", "http://localhost:9000/items", body)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", writer.FormDataContentType())

			// execute test target
			got, _, _, err := parseAddItemRequest(req)

			// confirm the result
			if err != nil {
				if !tt.err {
					t.Errorf("unexpected error: %v", err)
				}
				return
			}
			if diff := cmp.Diff(tt.wants.req, got); diff != "" {
				t.Errorf("unexpected request (-want +got):\n%s", diff)
			}
		})
	}
}

func TestHelloHandler(t *testing.T) {
	t.Parallel()

	// Please comment out for STEP 6-2
	// predefine what we want
	type wants struct {
		code int               // desired HTTP status code
		body map[string]string // desired body
	}
	want := wants{
		code: http.StatusOK,
		body: map[string]string{"message": "Hello, world!"},
	}

	// set up test
	req := httptest.NewRequest("GET", "/hello", nil)
	res := httptest.NewRecorder()

	h := &Handlers{}
	h.Hello(res, req)

	// STEP 6-2: confirm the status code
	if res.Code != want.code {
		t.Errorf("expected status code %d, got %d", want.code, res.Code)
	}

	// STEP 6-2: confirm response body
	gotBody := make(map[string]string)
	if err := json.Unmarshal(res.Body.Bytes(), &gotBody); err != nil {
		t.Fatalf("failed to unmarshal response body: %v", err)
	}

	if diff := cmp.Diff(want.body, gotBody); diff != "" {
		t.Errorf("unexpected body (-want +got):\n%s", diff)
	}
}

func TestAddItem(t *testing.T) {
	t.Parallel()

	type wants struct {
		code int
	}
	cases := map[string]struct {
		args     map[string]string
		injector func(m *MockItemRepository)
		wants
	}{
		"ok: correctly inserted": {
			args: map[string]string{
				"name":     "used iPhone 16e",
				"category": "phone",
				"image":    "dummy.png",
			},
			injector: func(m *MockItemRepository) {
				// STEP 6-3: define mock expectation
				// succeeded to insert
				m.EXPECT().GetCategoryID(gomock.Any(), "phone").Return(1, nil)
				m.EXPECT().Insert(gomock.Any(), gomock.Any()).Return(nil)
			},
			wants: wants{
				code: http.StatusOK,
			},
		},
		"ng: failed to insert": {
			args: map[string]string{
				"name":     "used iPhone 16e",
				"category": "phone",
				"image":    "dummy.png",
			},
			injector: func(m *MockItemRepository) {
				// STEP 6-3: define mock expectation
				// failed to insert
				m.EXPECT().GetCategoryID(gomock.Any(), "phone").Return(0, fmt.Errorf("category not found"))
			},
			wants: wants{
				code: http.StatusInternalServerError,
			},
		},
	}

	for name, tt := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockIR := NewMockItemRepository(ctrl)
			tt.injector(mockIR)
			h := &Handlers{itemRepo: mockIR}

			var b bytes.Buffer
			w := multipart.NewWriter(&b)
			for k, v := range tt.args {
				if k != "image" {
					_ = w.WriteField(k, v)
				}
			}

			//create the image for the test
			fileWriter, err := w.CreateFormFile("image", "dummy.png")
			if err != nil {
				t.Fatalf("failed to create form file: %v", err)
			}

			//send the image for the test
			dummyImage := []byte("dummy image data")
			_, err = fileWriter.Write(dummyImage)
			if err != nil {
				t.Fatalf("failed to write dummy imagedata: %v", err)
			}

			w.Close()

			req := httptest.NewRequest("POST", "/items", &b)
			req.Header.Set("Content-Type", w.FormDataContentType())

			rr := httptest.NewRecorder()
			h.AddItem(rr, req)

			if tt.wants.code != rr.Code {
				t.Errorf("expected status code %d, got %d", tt.wants.code, rr.Code)
			}
			if tt.wants.code >= 400 {
				return
			}

			for _, v := range tt.args {
				if !strings.Contains(rr.Body.String(), v) {
					t.Errorf("response body does not contain %s, got: %s", v, rr.Body.String())
				}
			}
		})
	}
}

// STEP 6-4: uncomment this test
func TestAddItemE2e(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test")
	}

	db, closers, err := setupDB(t)
	if err != nil {
		t.Fatalf("failed to set up database: %v", err)
	}
	t.Cleanup(func() {
		for _, c := range closers {
			c()
		}
	})

	type wants struct {
		code int
	}
	cases := map[string]struct {
		args map[string]string
		wants
	}{
		"ok: correctly inserted": {
			args: map[string]string{
				"name":     "used iPhone 16e",
				"category": "phone",
			},
			wants: wants{
				code: http.StatusOK,
			},
		},
		"ng: failed to insert": {
			args: map[string]string{
				"name":     "",
				"category": "phone",
			},
			wants: wants{
				code: http.StatusBadRequest,
			},
		},
	}

	for name, tt := range cases {
		t.Run(name, func(t *testing.T) {
			h := &Handlers{itemRepo: &itemRepository{db: db}}

			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			for k, v := range tt.args {
				_ = writer.WriteField(k, v)
			}
			fileWriter, _ := writer.CreateFormFile("image", "dummy.png")
			fileWriter.Write([]byte("duummy image content"))
			writer.Close()

			req := httptest.NewRequest("POST", "/items", &body)
			req.Header.Set("Content-Type", writer.FormDataContentType())

			rr := httptest.NewRecorder()
			h.AddItem(rr, req)

			// check response
			if tt.wants.code != rr.Code {
				t.Errorf("expected status code %d, got %d", tt.wants.code, rr.Code)
			}
			if tt.wants.code >= 400 {
				return
			}
			for _, v := range tt.args {
				if !strings.Contains(rr.Body.String(), v) {
					t.Errorf("response body does not contain %s, got: %s", v, rr.Body.String())
				}
			}

			// STEP 6-4: check inserted data
			var id int
			var name, category string
			err := db.QueryRow(`
				SELECT items.id, items.name, categories.name
				FROM items
				JOIN categories ON items.category_id = categories.id
				WHERE items.name = ?`, tt.args["name"]).Scan(&id, &name, &category)

			if err != nil {
				t.Fatalf("failed to retrieve inserted item: %v", err)
			}

			if name != tt.args["name"] || category != tt.args["category"] {
				t.Errorf("expected (name, category) = (%s, %s), but got (%s, %s)", tt.args["name"], tt.args["category"], name, category)
			}
		})
	}
}

func setupDB(t *testing.T) (db *sql.DB, closers []func(), e error) {
	t.Helper()

	defer func() {
		if e != nil {
			for _, c := range closers {
				c()
			}
		}
	}()

	// create a temporary file for e2e testing
	f, err := os.CreateTemp(".", "*.sqlite3")
	if err != nil {
		return nil, nil, err
	}
	closers = append(closers, func() {
		f.Close()
		os.Remove(f.Name())
	})

	// set up tables
	db, err = sql.Open("sqlite3", f.Name())
	if err != nil {
		return nil, nil, err
	}
	closers = append(closers, func() {
		db.Close()
	})

	// TODO: replace it with real SQL statements.
	cmd := `
	CREATE TABLE IF NOT EXISTS categories(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name VARCHAR(255) NOT NULL UNIQUE
	);
	
	CREATE TABLE IF NOT EXISTS items(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name VARCHAR(255) NOT NULL,
		category_id INTEGER NOT NULL,
		image_name VARCHAR(255) NOT NULL,
		FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE CASCADE
	);`

	_, err = db.Exec(cmd)
	if err != nil {
		return nil, nil, err
	}

	return db, closers, nil
}
