package db

import (
	"bytes"
	"encoding/json"
	"github.com/dyrkin/zigbee-steward/logger"
	"github.com/dyrkin/zigbee-steward/model"
	"github.com/natefinch/atomic"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
)

var log = logger.MustGetLogger("db")

type Devices map[string]*model.Device

type tables struct {
	Devices Devices
}

type Db struct {
	tables   *tables
	location string
}

var database *Db
var rw *sync.RWMutex

func (db *Db) Tables() *tables {
	return db.tables
}

func (devices Devices) Add(device *model.Device) {
	update(func() {
		database.tables.Devices[device.IEEEAddress] = device
	})
}

func (devices Devices) Remove(ieeeAddress string) {
	update(func() {
		delete(database.tables.Devices, ieeeAddress)
	})
}

func (devices Devices) Exists(ieeeAddress string) bool {
	_, ok := database.tables.Devices[ieeeAddress]
	return ok
}

func Database() *Db {
	return database
}

func init() {
	rw = &sync.RWMutex{}
	dbPath, err := filepath.Abs("db.json")
	if err != nil {
		log.Fatalf("Can't load database. %s", err)
	}
	database = &Db{
		tables: &tables{
			Devices: map[string]*model.Device{},
		},
		location: dbPath,
	}
	if !exists() {
		write()
	}
	read()
}

func update(updateFn func()) {
	rw.Lock()
	defer rw.Unlock()
	updateFn()
	write()
}

func read() {
	data, err := ioutil.ReadFile(database.location)
	if err != nil {
		log.Fatalf("Can't read database. %s", err)
	}
	if err = json.Unmarshal(data, &(database.tables)); err != nil {
		log.Fatalf("Can't unmarshal database. %s", err)
	}
}

func write() {
	data, err := json.MarshalIndent(database.tables, "", "    ")
	if err != nil {
		log.Fatalf("Can't marshal database. %s", err)
	}
	if err = atomic.WriteFile(database.location, bytes.NewBuffer(data)); err != nil {
		log.Fatalf("Can't write database. %s", err)
	}
	return
}

func exists() bool {
	_, err := os.Stat(database.location)
	return !os.IsNotExist(err)
}
