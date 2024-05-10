package storage

import (
	"log"
	"os"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type DBStorage struct {
	db *gorm.DB
}

func NewStorage(dbpath string, verbose bool, table ...any) (*DBStorage, error) {
	loglevel := logger.Error
	if verbose {
		loglevel = logger.Info
	}
	db, err := gorm.Open(sqlite.Open(dbpath), &gorm.Config{
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
	err = db.Migrator().AutoMigrate(table...)
	if err != nil {
		return nil, err
	}
	return &DBStorage{
		db: db,
	}, nil
}

func (d *DBStorage) GormDB() *gorm.DB {
	return d.db
}

func (d *DBStorage) Close() {
	sqlDB, err := d.db.DB()
	if err != nil {
		return
	}
	sqlDB.Close()
}
