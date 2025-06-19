// Package db provides functions for managing team data and progress using a BoltDB backend.
package db

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
	"os"
	"path/filepath"
	"time"

	"go.etcd.io/bbolt"
)

var (
	bucketTeams = []byte("teams")
)

var db *bbolt.DB

func Open(config Config) {
	if db != nil {
		panic("db: already opened")
	}
	if config.File == "" {
		panic("db: file is required")
	}

	err := os.MkdirAll(filepath.Dir(config.File), 0755)
	if err != nil {
		panic(fmt.Errorf("db: create db dir: %w", err))
	}

	db, err = bbolt.Open(config.File, 0600, &bbolt.Options{
		Timeout: 30 * time.Second,
	})
	if err != nil {
		panic(fmt.Errorf("db: open bbolt db: %w", err))
	}

	err = db.Update(func(tx *bbolt.Tx) error {
		for _, bucket := range [][]byte{
			bucketTeams,
		} {
			_, err := tx.CreateBucketIfNotExists(bucket)
			if err != nil {
				return fmt.Errorf("create bucket %q: %w", bucket, err)
			}
		}

		return nil
	})
	if err != nil {
		db.Close()
		panic(fmt.Errorf("db: initialize buckets: %w", err))
	}
}

func Close() error {
	if db == nil {
		panic("db: not opened")
	}

	err := db.Close()
	if err != nil {
		return fmt.Errorf("db: close bbolt db: %w", err)
	}
	db = nil
	return nil
}

type closerFunc func() error

func (f closerFunc) Close() error {
	return f()
}

func Closer() io.Closer {
	return closerFunc(Close)
}

func Teams() ([]string, error) {
	if db == nil {
		panic("db: not opened")
	}

	var teams []string
	err := db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketTeams)
		if b == nil {
			return fmt.Errorf("db: teams bucket not found")
		}

		return b.ForEach(func(k, v []byte) error {
			teams = append(teams, string(k))
			return nil
		})
	})

	if err != nil {
		return nil, fmt.Errorf("db: get teams: %w", err)
	}

	return teams, nil
}

type TeamProgress struct {
	Puzzles map[string]PuzzleProgress `json:"puzzles"`
}

type PuzzleProgress struct {
	FirstOpened time.Time `json:"first_opened"`
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(fmt.Errorf("db: must: %w", err))
	}
	return v
}

func AddTeamProgress(team, puzzle string, time time.Time) error {
	if db == nil {
		panic("db: not opened")
	}

	return db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketTeams)
		if b == nil {
			return fmt.Errorf("db: teams bucket not found")
		}

		var progress TeamProgress

		data := b.Get([]byte(team))
		if data == nil {
			progress = TeamProgress{
				Puzzles: make(map[string]PuzzleProgress, 1),
			}
		} else {
			err := json.Unmarshal(data, &progress)
			if err != nil {
				return fmt.Errorf("db: unmarshal team progress for %q: %w", team, err)
			}
		}

		if _, exists := progress.Puzzles[puzzle]; exists {
			return nil
		}

		progress.Puzzles[puzzle] = PuzzleProgress{
			FirstOpened: time,
		}
		return b.Put([]byte(team), must(json.Marshal(progress)))
	})
}

var errStop = fmt.Errorf("stop iteration")

func All() iter.Seq2[string, TeamProgress] {
	if db == nil {
		panic("db: not opened")
	}

	return func(yield func(string, TeamProgress) bool) {
		err := db.View(func(tx *bbolt.Tx) error {
			b := tx.Bucket(bucketTeams)
			if b == nil {
				return fmt.Errorf("db: teams bucket not found")
			}

			return b.ForEach(func(k, v []byte) error {
				var progress TeamProgress
				err := json.Unmarshal(v, &progress)
				if err != nil {
					return fmt.Errorf("db: unmarshal team progress for %q: %w", k, err)
				}

				if !yield(string(k), progress) {
					return errStop
				}
				return nil
			})
		})

		if err != nil {
			if errors.Is(err, errStop) {
				return
			}
			panic(fmt.Errorf("db: get all teams: %w", err))
		}
	}
}
