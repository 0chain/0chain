package event

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"0chain.net/core/config"
	"0chain.net/smartcontract/dbs/goose"
	"0chain.net/smartcontract/dbs/postgresql"
	"github.com/0chain/common/core/logging"
	_ "github.com/jackc/pgx/v5"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var gEventDB *EventDb

// returns an event db transaction and clean up function
func GetTestEventDB(t *testing.T) (*EventDb, func()) {
	db, err := gEventDB.Begin(context.Background())
	require.NoError(t, err)

	db.managePermanentPartitions(0)
	db.managePartitions(0)

	return db, func() {
		db.Rollback()
	}
}

func TestMain(m *testing.M) {
	logging.InitLogging("development", "")
	var db *sql.DB
	// uses a sensible default on windows (tcp/http) and linux/osx (socket)
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not construct pool: %s", err)
	}

	err = pool.Client.Ping()
	if err != nil {
		log.Fatalf("Could not connect to Docker: %s", err)
	}

	// pulls an image, creates a container based on it and runs it
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "14",
		Env: []string{
			"POSTGRES_PASSWORD=zchian",
			"POSTGRES_USER=zchain_user",
			"POSTGRES_DB=events_db",
			"listen_addresses = '*'",
		},
	}, func(config *docker.HostConfig) {
		// set AutoRemove to true so that stopped container goes away by itself
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}

	hostAndPort := getHostPort(resource, "5432/tcp")
	parts := strings.Split(hostAndPort, ":")
	host := parts[0]
	port := parts[1]
	databaseUrl := fmt.Sprintf("postgres://zchain_user:zchian@%s/events_db?sslmode=disable", hostAndPort)

	log.Println("Connecting to database on url:", databaseUrl)

	resource.Expire(600) // Tell docker to hard kill the container in 600 seconds (10 Mins)

	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	pool.MaxWait = 120 * time.Second
	if err = pool.Retry(func() error {
		db, err = sql.Open("pgx", databaseUrl)
		if err != nil {
			return err
		}
		return db.Ping()
	}); err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{})
	if err != nil {
		log.Fatalf("Could not connect to gorm: %s", err)
	}

	dbSetting := config.DbSettings{
		AggregatePeriod:                10,
		PartitionKeepCount:             10,
		PartitionChangePeriod:          100,
		PermanentPartitionChangePeriod: 20,
		PermanentPartitionKeepCount:    1,
		PageLimit:                      10,
	}

	config.Configuration().ChainConfig = &TestConfig{conf: &TestConfigData{DbsSettings: dbSetting}}

	access := config.DbAccess{
		Enabled:         true,
		Name:            "events_db",
		User:            "zchain_user",
		Password:        "zchian",
		Host:            host,
		Port:            port,
		MaxIdleConns:    100,
		MaxOpenConns:    200,
		ConnMaxLifetime: 20 * time.Second,
		Slowtablespace:  "hdd_tablespace",
	}
	gEventDB = &EventDb{
		Store:         postgresql.New(gormDB),
		dbConfig:      access,
		eventsChannel: make(chan BlockEvents, 1),
		settings:      dbSetting,
	}

	s, err := gormDB.DB()
	if err != nil {
		log.Fatal(err)
	}
	goose.Init()
	goose.Migrate(s)

	//Run tests
	code := m.Run()

	// You can't defer this because os.Exit doesn't care for defer
	if err := pool.Purge(resource); err != nil {
		log.Fatalf("Could not purge resource: %s", err)
	}

	os.Exit(code)
}

func getHostPort(resource *dockertest.Resource, id string) string {
	dockerURL := os.Getenv("DOCKER_HOST_ENV")
	if dockerURL == "" {
		return resource.GetHostPort(id)
	}
	return dockerURL + ":" + resource.GetPort(id)
}
