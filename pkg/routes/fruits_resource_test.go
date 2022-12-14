package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/kameshsampath/go-fruits-api/pkg/db"
	"github.com/kameshsampath/go-fruits-api/pkg/utils"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	"github.com/uptrace/bun/dbfixture"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
)

var (
	log *logrus.Logger
)

func init() {
	os.Remove(getDBFile("test"))
}

func getDBFile(dbName string) string {
	cwd, _ := os.Getwd()
	return path.Join(cwd, "testdata", fmt.Sprintf("%s.db", dbName))
}

func loadFixtures(ctx context.Context) (*db.Config, error) {
	log = utils.LogSetup(os.Stdout, utils.LookupEnvOrString("TEST_LOG_LEVEL", "info"))
	dbt := utils.LookupEnvOrString("FRUITS_DB_TYPE", "sqlite")
	var dbc *db.Config
	if dbt == "sqlite" {
		dbc = db.New(
			db.WithLogger(log),
			db.WithDBType(dbt),
			db.WithDBFile("testdata/test.db"))
	} else {
		dbc = db.New(
			db.WithLogger(log),
			db.WithDBType(dbt))
	}
	dbc.Init(ctx)

	if err := dbc.DB.Ping(); err != nil {
		return nil, err
	}

	dbc.DB.RegisterModel((*db.Fruit)(nil))
	dbfx := dbfixture.New(dbc.DB, dbfixture.WithRecreateTables())
	if err := dbfx.Load(ctx, os.DirFS("."), "testdata/fixtures.yaml"); err != nil {
		return nil, err
	}

	return dbc, nil
}

func TestAddFruit(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dbc, err := loadFixtures(ctx)
	if err != nil {
		t.Fatal(err)
	}

	testCases := map[string]struct {
		requestBody string
		statusCode  int
		want        db.Fruit
	}{
		"withoutId": {
			requestBody: `{
        "name": "Test Fruit 2",
        "season": "Spring"
        }`,
			statusCode: http.StatusCreated,
			want: db.Fruit{
				ID:     10,
				Name:   "Test Fruit 2",
				Season: "Spring",
			},
		},
		"withId": {
			requestBody: `{
        "id": 11,
        "name": "Test Fruit",
        "season": "Summer"
        }`,
			statusCode: http.StatusCreated,
			want: db.Fruit{
				ID:     11,
				Name:   "Test Fruit",
				Season: "Summer",
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			var got db.Fruit
			e := echo.New()
			req := httptest.NewRequest(http.MethodPost, "/api/fruits/add", strings.NewReader(tc.requestBody))
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			ep := &Endpoints{
				Config: dbc,
			}
			if c := e.NewContext(req, rec); assert.NoError(t, ep.AddFruit(c)) {
				assert.Equal(t, tc.statusCode, rec.Code)
				dbConn := ep.Config.DB
				err := dbConn.NewSelect().
					Model(&got).
					Where("name = ?", tc.want.Name).
					Scan(ctx)
				if err != nil {
					t.Fatal(err)
				}
				//Verify Fruit
				if diff := cmp.Diff(tc.want, got, cmpopts.IgnoreFields(db.Fruit{}, "CreatedAt", "ModifiedAt")); diff != "" {
					t.Errorf("AddFruit() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestDeleteAllFruit(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	dbc, err := loadFixtures(ctx)
	if err != nil {
		t.Fatal(err)
	}
	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/fruits/", nil)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	ep := &Endpoints{
		Config: dbc,
	}
	if assert.NoError(t, ep.DeleteAll(c)) {
		assert.Equal(t, http.StatusNoContent, rec.Code)
		dbConn := ep.Config.DB
		c, err := dbConn.NewSelect().
			Model((*db.Fruit)(nil)).
			Count(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		assert.NoError(t, err)
		assert.Truef(t, c == 0, "Expecting no Fruits but it does")
	}
}

func TestDeleteFruit(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	fruitID := "5"
	dbc, err := loadFixtures(ctx)
	if err != nil {
		t.Fatal(err)
	}
	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/fruits/:id", nil)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/api/fruits/:id")
	c.SetParamNames("id")
	c.SetParamValues(fruitID)
	ep := &Endpoints{
		Config: dbc,
	}
	if assert.NoError(t, ep.DeleteFruit(c)) {
		assert.Equal(t, http.StatusNoContent, rec.Code)
		dbConn := ep.Config.DB
		ID, _ := strconv.Atoi(fruitID)
		exists, err := dbConn.NewSelect().
			Model(&db.Fruit{ID: ID}).
			WherePK().
			Exists(ctx)
		if err != nil {
			t.Fatal(err)
		}
		assert.Falsef(t, exists, "Expecting Fruit with ID %s not to exist but it does", fruitID)
	}
}

func TestGetFruitByName(t *testing.T) {
	testCases := map[string]struct {
		name string
		want db.Fruits
	}{
		"default": {
			name: "Apple",
			want: db.Fruits{
				{
					ID:     8,
					Name:   "Apple",
					Emoji:  "U+1F34E",
					Season: "Fall",
				},
			},
		},
		"allLower": {
			name: "apple",
			want: db.Fruits{
				{
					ID:     8,
					Name:   "Apple",
					Emoji:  "U+1F34E",
					Season: "Fall",
				},
			},
		},
		"allCaps": {
			name: "APPLE",
			want: db.Fruits{
				{
					ID:     8,
					Name:   "Apple",
					Emoji:  "U+1F34E",
					Season: "Fall",
				},
			},
		},
		"mixedCase": {
			name: "apPLe",
			want: db.Fruits{
				{
					ID:     8,
					Name:   "Apple",
					Emoji:  "U+1F34E",
					Season: "Fall",
				},
			},
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	dbc, err := loadFixtures(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/api/fruits/:name", nil)
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPath("/api/fruits/:name")
			c.SetParamNames("name")
			c.SetParamValues(tc.name)
			ep := &Endpoints{
				Config: dbc,
			}
			if assert.NoError(t, ep.GetFruitsByName(c)) {
				assert.Equal(t, http.StatusOK, rec.Code)
				var got db.Fruits
				b := rec.Body.Bytes()
				err := json.Unmarshal(b, &got)
				if err != nil {
					t.Fatal(err)
				}
				assert.NotNil(t, got, "Expecting the response to have Fruit(s) object but got none")
				//Verify Fruit
				if diff := cmp.Diff(tc.want, got, cmpopts.IgnoreFields(db.Fruit{}, "CreatedAt", "ModifiedAt")); diff != "" {
					t.Errorf("GetFruitsByName() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestGetFruitsBySeason(t *testing.T) {
	testCases := map[string]struct {
		season string
		want   db.Fruits
	}{
		"default": {
			season: "Summer",
			want: db.Fruits{
				{
					ID:     5,
					Name:   "Blueberry",
					Emoji:  "U+1FAD0",
					Season: "Summer",
				},
				{
					ID:     6,
					Name:   "Banana",
					Emoji:  "U+1F34C",
					Season: "Summer",
				},
				{
					ID:     7,
					Name:   "Watermelon",
					Emoji:  "U+1F349",
					Season: "Summer",
				},
			},
		},
		"allLower": {
			season: "summer",
			want: db.Fruits{
				{
					ID:     5,
					Name:   "Blueberry",
					Emoji:  "U+1FAD0",
					Season: "Summer",
				},
				{
					ID:     6,
					Name:   "Banana",
					Emoji:  "U+1F34C",
					Season: "Summer",
				},
				{
					ID:     7,
					Name:   "Watermelon",
					Emoji:  "U+1F349",
					Season: "Summer",
				},
			},
		},
		"allCaps": {
			season: "SUMMER",
			want: db.Fruits{
				{
					ID:     5,
					Name:   "Blueberry",
					Emoji:  "U+1FAD0",
					Season: "Summer",
				},
				{
					ID:     6,
					Name:   "Banana",
					Emoji:  "U+1F34C",
					Season: "Summer",
				},
				{
					ID:     7,
					Name:   "Watermelon",
					Emoji:  "U+1F349",
					Season: "Summer",
				},
			},
		},
		"mixedCase": {
			season: "suMMEr",
			want: db.Fruits{
				{
					ID:     5,
					Name:   "Blueberry",
					Emoji:  "U+1FAD0",
					Season: "Summer",
				},
				{
					ID:     6,
					Name:   "Banana",
					Emoji:  "U+1F34C",
					Season: "Summer",
				},
				{
					ID:     7,
					Name:   "Watermelon",
					Emoji:  "U+1F349",
					Season: "Summer",
				},
			},
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	dbc, err := loadFixtures(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/api/fruits/:season", nil)
			req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetPath("/api/fruits/:season")
			c.SetParamNames("season")
			c.SetParamValues(tc.season)
			ep := &Endpoints{
				Config: dbc,
			}
			if assert.NoError(t, ep.GetFruitsBySeason(c)) {
				assert.Equal(t, http.StatusOK, rec.Code)
				var got db.Fruits
				b := rec.Body.Bytes()
				err := json.Unmarshal(b, &got)
				if err != nil {
					t.Fatal(err)
				}
				assert.NotNil(t, got, "Expecting the response to have Fruits object but got none")
				//Verify Fruit
				if diff := cmp.Diff(tc.want, got, cmpopts.IgnoreFields(db.Fruit{}, "CreatedAt", "ModifiedAt")); diff != "" {
					t.Errorf("GetFruitsBySeason() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestGetAllFruits(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	dbc, err := loadFixtures(ctx)
	if err != nil {
		t.Fatal(err)
	}
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/fruits", nil)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	ep := &Endpoints{
		Config: dbc,
	}
	var want db.Fruits
	cwd, _ := os.Getwd()
	bw, err := os.ReadFile(path.Join(cwd, "testdata", "list.json"))
	if err != nil {
		t.Fatal(err)
	}
	err = json.Unmarshal(bw, &want)
	if err != nil {
		t.Fatal(err)
	}
	sort.Sort(want)
	if c := e.NewContext(req, rec); assert.NoError(t, ep.ListFruits(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
		var got db.Fruits
		b := rec.Body.Bytes()
		err := json.Unmarshal(b, &got)
		if err != nil {
			t.Fatal(err)
		}
		assert.NotNil(t, got, "Expecting the response to have Fruits object but got none")
		sort.Sort(got)
		//Verify Fruits
		if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(db.Fruit{}, "CreatedAt", "ModifiedAt")); diff != "" {
			t.Errorf("GetAllFruits() mismatch (-want +got):\n%s", diff)
		}
	}
}
