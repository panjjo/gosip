package db

import (
	"errors"

	"github.com/panjjo/gorm"
)

type Tx struct {
	db       *gorm.DB
	isCommit bool
}

func NewTx(db *gorm.DB) (*Tx, error) {
	tx := db.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	return &Tx{
		db: tx,
	}, nil
}

func (tx *Tx) End() error {
	if tx.isCommit {
		return nil
	}

	return tx.db.Rollback().Error
}

func (tx *Tx) Commit() error {

	err := tx.db.Commit().Error
	if err != nil {
		return err
	}
	tx.isCommit = true
	return nil
}

func (tx *Tx) DB() *gorm.DB {
	return tx.db
}

type TxB struct {
	dbs      []*gorm.DB
	isCommit bool
}

func NewTxB(db ...*gorm.DB) (*TxB, error) {
	dbs := []*gorm.DB{}
	for i := range db {
		tx := db[i].Begin()
		if tx.Error != nil {
			for o := range dbs {
				dbs[o].Rollback()
			}
			return nil, tx.Error
		}
		dbs = append(dbs, tx)
	}
	return &TxB{
		dbs: dbs,
	}, nil
}

func (tx *TxB) End() error {
	if tx.isCommit {
		return nil
	}
	es := ""

	for o := range tx.dbs {
		if err := tx.dbs[o].Rollback().Error; err != nil {
			es += err.Error()
		}
	}
	return errors.New(es)
}

func (tx *TxB) Commit() error {

	tx.isCommit = true

	for o := range tx.dbs {
		if err := tx.dbs[o].Commit().Error; err != nil {
			es := err.Error()
			for n := range tx.dbs[o:] {
				if err := tx.dbs[n+o].Rollback().Error; err != nil {
					es += err.Error()
				}
			}
			return errors.New(es)
		}
	}

	return nil
}

func (tx *TxB) DB(i int) *gorm.DB {
	return tx.dbs[i]
}

type TxF struct {
	db    *gorm.DB
	fs    []TxFunc
	retry int
}

type TxFunc = func(*gorm.DB) error

func NewTxF(db *gorm.DB, fs ...TxFunc) *TxF {
	return NewTxFR(db, 3, fs...)
}

func NewTxFR(db *gorm.DB, retry int, fs ...TxFunc) *TxF {
	return &TxF{
		db:    db,
		fs:    fs,
		retry: retry,
	}
}

func (tx *TxF) Add(f ...TxFunc) {
	tx.fs = append(tx.fs, f...)
}

func (tx *TxF) Commit() error {
	var err error
	for i := 0; i < tx.retry; i++ {
		if err = tx.commit(); err == nil {
			return nil
		}
	}
	return err
}

func (tx *TxF) commit() error {
	db := tx.db.Begin()
	if db.Error != nil {
		return db.Error
	}
	commit := false
	defer func() {
		if !commit {
			db.Rollback()
		}

	}()
	for _, v := range tx.fs {
		if err := v(db); err != nil {
			return err
		}
	}

	err := db.Commit().Error
	if err != nil {
		return err
	}
	commit = true
	return nil
}
