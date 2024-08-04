package postgres

import (
	"database/sql"
	"errors"

	"github.com/GLCharge/otelzap"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

func rollback(tx *sqlx.Tx, log *otelzap.Logger) {
	err := tx.Rollback()
	if err != nil && !errors.Is(err, sql.ErrTxDone) {
		log.Error("Failed to rollback transaction", zap.Error(err))
	}
}
