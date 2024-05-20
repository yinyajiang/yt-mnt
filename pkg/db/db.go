package db

import (
	"log"
	"os"
	"time"

	"path/filepath"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type DBStorage struct {
	db         *gorm.DB
	notCloseDB bool
}

type DBOption struct {
	DBPath        string
	OutDB         *gorm.DB
	NotCloseOutDB bool
}

func NewStorage(opt DBOption, isVerbose bool, table ...any) (*DBStorage, error) {
	if opt.DBPath != "" {
		loglevel := logger.Error
		if isVerbose {
			loglevel = logger.Info
		}
		os.MkdirAll(filepath.Dir(opt.DBPath), os.ModePerm)

		db, err := gorm.Open(sqlite.Open(opt.DBPath), &gorm.Config{
			Logger: logger.New(
				log.New(os.Stdout, "\r\n", log.LstdFlags),
				logger.Config{
					SlowThreshold: time.Millisecond * 100,
					LogLevel:      loglevel,
				},
			),
			PrepareStmt: true,
		})
		if err != nil {
			return nil, err
		}
		opt.OutDB = db
	}
	err := opt.OutDB.Migrator().AutoMigrate(table...)
	if err != nil {
		return nil, err
	}
	return &DBStorage{
		db:         opt.OutDB,
		notCloseDB: opt.NotCloseOutDB,
	}, nil
}

func (d *DBStorage) GormDB() *gorm.DB {
	return d.db
}

func (d *DBStorage) Close() {
	if d.notCloseDB {
		return
	}
	sqlDB, err := d.db.DB()
	if err != nil {
		return
	}
	sqlDB.Close()
}
