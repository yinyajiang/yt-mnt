package db

import (
	"log"
	"os"
	"sync"
	"time"

	"path/filepath"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type DBStorage struct {
	db         *gorm.DB
	notCloseDB bool

	_isClosed       bool
	_dbWriteOperate sync.WaitGroup
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
	d._isClosed = true
	d._dbWriteOperate.Wait()

	sqlDB, err := d.db.DB()
	if err != nil {
		return
	}
	sqlDB.Close()
}

func (d *DBStorage) IsClosed() bool {
	return d._isClosed
}

func (d *DBStorage) Delete(value interface{}, conds ...interface{}) error {
	d._dbWriteOperate.Add(1)
	defer d._dbWriteOperate.Done()
	return d.db.Unscoped().Delete(value, conds...).Error
}

func (d *DBStorage) DeleteAll(value interface{}) error {
	d._dbWriteOperate.Add(1)
	defer d._dbWriteOperate.Done()
	return d.db.Unscoped().Where("1 = 1").Delete(value).Error
}

func (d *DBStorage) Save(value interface{}) error {
	d._dbWriteOperate.Add(1)
	defer d._dbWriteOperate.Done()
	return d.db.Save(value).Error
}

func (d *DBStorage) Create(value interface{}) error {
	d._dbWriteOperate.Add(1)
	defer d._dbWriteOperate.Done()
	return d.db.Create(value).Error
}

func (d *DBStorage) Updates(values interface{}) error {
	d._dbWriteOperate.Add(1)
	defer d._dbWriteOperate.Done()
	return d.db.Updates(values).Error
}

func (d *DBStorage) WhereUpdates(where interface{}, values interface{}) error {
	d._dbWriteOperate.Add(1)
	defer d._dbWriteOperate.Done()
	return d.db.Where(where).Updates(values).Error
}
