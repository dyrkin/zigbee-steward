package db

import (
	"encoding/json"
	"github.com/dyrkin/zigbee-steward/logger"
	"github.com/dyrkin/zigbee-steward/model"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
)

var log = logger.MustGetLogger("db")

type tables struct {
	Devices map[string]*model.Device
}

type Db struct {
	Tables   *tables
	rw       *sync.RWMutex
	location string
}

var Database *Db

func init() {
	dbPath, err := filepath.Abs("db.json")
	if err != nil {
		log.Fatalf("Can't load database. %s", err)
	}

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		Database = &Db{
			Tables: &tables{
				Devices: map[string]*model.Device{},
			},
			rw:       &sync.RWMutex{},
			location: dbPath,
		}
		writeDatabase()
	}

	Database = &Db{
		Tables: &tables{
			Devices: map[string]*model.Device{},
		},
		rw:       &sync.RWMutex{},
		location: dbPath,
	}
	data, err := ioutil.ReadFile(dbPath)
	if err != nil {
		log.Fatalf("Can't read database. %s", err)
	}
	if err = json.Unmarshal(data, &(Database.Tables)); err != nil {
		log.Fatalf("Can't unmarshal database. %s", err)
	}
}

func writeDatabase() {
	data, err := json.MarshalIndent(Database.Tables, "", "    ")
	if err != nil {
		log.Fatalf("Can't marshal database. %s", err)
	}
	if err = ioutil.WriteFile(Database.location, data, 0644); err != nil {
		log.Fatalf("Can't write database. %s", err)
	}
	return
}

func (db *Db) AddDevice(device *model.Device) {
	db.rw.Lock()
	defer db.rw.Unlock()

	Database.Tables.Devices[device.IEEEAddress] = device
	writeDatabase()
}
