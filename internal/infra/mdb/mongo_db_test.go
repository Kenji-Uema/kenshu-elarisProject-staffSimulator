package mdb

import (
	"testing"

	"github.com/Kenji-Uema/staffSimulator/internal/config"
)

func TestBuildMongoURI(t *testing.T) {
	t.Parallel()

	t.Run("builds uri from host and credentials", func(t *testing.T) {
		cfg := config.MongoConfig{}
		cfg.Conn.Username = "user"
		cfg.Conn.Password = "pass"
		cfg.Conn.Host = "localhost:27017/db"

		got := buildMongoURI(cfg)
		want := "mongodb://user:pass@localhost:27017/db"
		if got != want {
			t.Fatalf("buildMongoURI() = %q, want %q", got, want)
		}
	})

	t.Run("uses full mongodb uri host as is", func(t *testing.T) {
		cfg := config.MongoConfig{}
		cfg.Conn.Username = "ignored"
		cfg.Conn.Password = "ignored"
		cfg.Conn.Host = "mongodb://root:secret@localhost:27017/?authSource=admin"

		got := buildMongoURI(cfg)
		want := "mongodb://root:secret@localhost:27017/?authSource=admin"
		if got != want {
			t.Fatalf("buildMongoURI() = %q, want %q", got, want)
		}
	})
}
