package dbtransaction

import (
	"context"
	"database/sql"
)

type DbTransactionUtil interface {
	InitTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
	CommitTx(tx *sql.Tx) error
	RollbackTx(tx *sql.Tx) error
	RollbackOrCommitTx(tx *sql.Tx, err error) error
}

type dbTransactionUtil struct {
	db *sql.DB
}

func NewDbTransactionUtil(db *sql.DB) DbTransactionUtil {
	return &dbTransactionUtil{
		db: db,
	}
}

func (r *dbTransactionUtil) InitTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	tx, err := r.db.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}

	return tx, nil
}

func (r *dbTransactionUtil) RollbackTx(tx *sql.Tx) error {
	err := tx.Rollback()
	if err != nil {
		return err
	}
	return nil
}


func (r *dbTransactionUtil) CommitTx(tx *sql.Tx) (error) {
	err := tx.Commit()
	if err != nil {
		return err
	}
	return nil
}

func (r *dbTransactionUtil) RollbackOrCommitTx(tx *sql.Tx, err error) (error) {
	if err != nil {
		rollbackErr := r.RollbackTx(tx)
		if rollbackErr != nil {
			return rollbackErr
		}
		return err
	}

	commitErr := r.CommitTx(tx)
	if commitErr != nil {
		return commitErr
	}

	return nil
}
