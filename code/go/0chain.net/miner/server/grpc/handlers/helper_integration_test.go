package handlers

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"0chain.net/core/viper"

	"google.golang.org/grpc"
	"gorm.io/driver/postgres"

	minerproto "0chain.net/miner/proto/api/src/proto"
	"github.com/0chain/blobber/code/go/0chain.net/blobbercore/config"

	"testing"

	"gorm.io/gorm"
)

const BlobberTestAddr = "localhost:~~TO_DO~~"
const RetryAttempts = 8
const RetryTimeout = 3

type TestDataController struct {
	db *gorm.DB
}

func setupHandlerIntegrationTests(t *testing.T) (minerproto.MinerServiceClient, *TestDataController) {
	args := make(map[string]bool)
	for _, arg := range os.Args {
		args[arg] = true
	}
	if !args["integration"] {
		t.Skip()
	}

	var conn *grpc.ClientConn
	var err error
	for i := 0; i < RetryAttempts; i++ {
		conn, err = grpc.Dial(BlobberTestAddr, grpc.WithInsecure())
		if err != nil {
			log.Println(err)
			<-time.After(time.Second * RetryTimeout)
			continue
		}
		break
	}
	if err != nil {
		t.Fatal(err)
	}
	bClient := minerproto.NewMinerServiceClient(conn)

	setupIntegrationTestConfig(t)
	db, err := gorm.Open(postgres.Open(fmt.Sprintf(
		"host=%v port=%v user=%v dbname=%v password=%v sslmode=disable",
		config.Configuration.DBHost, config.Configuration.DBPort,
		config.Configuration.DBUserName, config.Configuration.DBName,
		config.Configuration.DBPassword)), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	tdController := NewTestDataController(db)

	return bClient, tdController
}

func setupIntegrationTestConfig(t *testing.T) {

	pwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	configDir := strings.Split(pwd, "/code/go")[0] + "/config"
	config.SetupDefaultConfig()
	config.SetupConfig(configDir)

	config.Configuration.DBHost = "localhost"
	config.Configuration.DBName = viper.GetString("db.name")
	config.Configuration.DBPort = viper.GetString("db.port")
	config.Configuration.DBUserName = viper.GetString("db.user")
	config.Configuration.DBPassword = viper.GetString("db.password")
}

func NewTestDataController(db *gorm.DB) *TestDataController {
	return &TestDataController{db: db}
}
