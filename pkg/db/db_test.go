package db

// import (
// 	"context"
// 	"os"
// 	"testing"
// 	"time"

// 	"github.com/kameshsampath/go-fruits-api/pkg/utils"
// 	"github.com/stretchr/testify/assert"
// 	"github.com/uptrace/bun"
// 	"github.com/uptrace/bun/dbfixture"
// 	"github.com/uptrace/bun/dialect"
// )

// func TestInitDB(t *testing.T) {
// 	log := utils.LogSetup(os.Stdout, utils.LookupEnvOrString("TEST_LOG_LEVEL", "info"))
// 	dbt := utils.LookupEnvOrString("FRUITS_DB_TYPE", "sqlite")
// 	var dbc *Config
// 	if dbt == "sqlite" {
// 		dbc = New(
// 			WithLogger(log),
// 			WithDBType(dbt),
// 			WithDBFile("testdata/test.db"))
// 	} else {
// 		dbc = New(
// 			WithLogger(log),
// 			WithDBType(dbt))
// 	}

// 	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
// 	defer cancel()

// 	dbc.Init(ctx)

// 	err := dbc.DB.Ping()

// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	dbc.DB.RegisterModel((*Fruit)(nil))

// 	dbfx := dbfixture.New(dbc.DB, dbfixture.WithRecreateTables())
// 	if err := dbfx.Load(ctx, os.DirFS("."), "testdata/fixtures.yaml"); err != nil {
// 		t.Fatalf("Unable to load fixtures, %s", err)
// 	}

// 	expected := dbfx.MustRow("Fruit.mango").(*Fruit)
// 	assert.NotNil(t, expected)

// 	actual := &Fruit{}

// 	err = dbc.DB.NewSelect().
// 		Model(actual).
// 		Where("? = ?", bun.Ident("name"), "Mango").
// 		Scan(context.Background())
// 	assert.NoError(t, err)

// 	assert.Equal(t, 1, 1, "Expected ID to be  %d but got %d", 1, actual.ID)
// 	assert.Equal(t, expected.Name, actual.Name, "Expected Name to be  %s but got %s", expected.Name, actual.Name)
// 	assert.Equal(t, expected.Name, actual.Name, "Expected Season to be  %s but got %s", expected.Season, actual.Season)

// 	var lastID int
// 	var seqQuery string
// 	switch dbc.DBType {
// 	case dialect.PG:
// 		seqQuery = "SELECT currval(pg_get_serial_sequence('fruits','id'))"
// 	case dialect.MySQL:
// 		seqQuery = "SELECT LAST_INSERT_ID()"
// 	default:
// 		seqQuery = "SELECT ROWID from FRUITS order by ROWID DESC limit 1"
// 	}

// 	err = dbc.DB.NewRaw(seqQuery).Scan(context.Background(), &lastID)
// 	assert.NoError(t, err)

// 	assert.Equal(t, 1, 1, "Expected Last Sequential ID to be  %d but got %d", 9, lastID)

// 	if dbc.DBType == dialect.SQLite {
// 		tearDown()
// 	}
// }

// func tearDown() {
// 	os.Remove("testdata/test.db")
// }
