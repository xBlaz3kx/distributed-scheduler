package database

import (
	"context"
	"net/url"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

// Config is the required properties to use the database.
type Config struct {
	User         string `conf:"default:scheduler" mapstructure:"user" yaml:"user"`
	Password     string `conf:"default:scheduler,mask" mapstructure:"password" yaml:"password"`
	Host         string `conf:"default:localhost:5436" mapstructure:"host" yaml:"host"`
	Name         string `conf:"default:scheduler" mapstructure:"name" yaml:"name"`
	MaxIdleConns int    `conf:"default:3" mapstructure:"max_idle_conns" yaml:"maxIdleConns"`
	MaxOpenConns int    `conf:"default:2" mapstructure:"max_open_conns" yaml:"maxOpenConns"`
	DisableTLS   bool   `conf:"default:true" mapstructure:"disable_tls" yaml:"disableTLS"`
}

// Open knows how to open a database connection based on the configuration.
func Open(cfg Config) (*sqlx.DB, error) {
	sslMode := "require"
	if cfg.DisableTLS {
		sslMode = "disable"
	}

	q := make(url.Values)
	q.Set("sslmode", sslMode)
	q.Set("timezone", "utc")

	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(cfg.User, cfg.Password),
		Host:     cfg.Host,
		Path:     cfg.Name,
		RawQuery: q.Encode(),
	}

	db, err := sqlx.Open("pgx", u.String())
	if err != nil {
		return nil, err
	}
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetMaxOpenConns(cfg.MaxOpenConns)

	return db, nil
}

// StatusCheck returns nil if it can successfully talk to the database. It
// returns a non-nil error otherwise.
func StatusCheck(ctx context.Context, db *sqlx.DB) error {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Second)
		defer cancel()
	}

	var pingError error
	for attempts := 1; ; attempts++ {
		pingError = db.Ping()
		if pingError == nil {
			break
		}
		time.Sleep(time.Duration(attempts) * 100 * time.Millisecond)
		if ctx.Err() != nil {
			return ctx.Err()
		}
	}

	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Run a simple query to determine connectivity.
	// Running this query forces a round trip through the database.
	const q = `SELECT true`
	var tmp bool
	return db.QueryRowContext(ctx, q).Scan(&tmp)
}

type HealthcheckAdapter struct {
	DB *sqlx.DB
}

func NewHealthChecker(db *sqlx.DB) *HealthcheckAdapter {
	return &HealthcheckAdapter{DB: db}
}

func (h *HealthcheckAdapter) Pass() bool {
	return StatusCheck(context.Background(), h.DB) == nil
}

func (h *HealthcheckAdapter) Name() string {
	return "postgres"
}
