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
	bucketTeams   = []byte("teams")
	bucketRenames = []byte("renames")
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
			bucketRenames,
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
	var teams []string
	for team, progress := range AllTeams() {
		if progress.Hidden {
			continue
		}
		teams = append(teams, team)
	}
	return teams, nil
}

type TeamProgress struct {
	Puzzles map[string]PuzzleProgress `json:"puzzles"`
	Hidden  bool                      `json:"hidden"`
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

func modify(team string, modify func(*TeamProgress, bool) (*TeamProgress, error)) error {
	if db == nil {
		panic("db: not opened")
	}

	return db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketTeams)
		if b == nil {
			return fmt.Errorf("db: teams bucket not found")
		}

		var progress *TeamProgress
		exists := false

		data := b.Get([]byte(team))
		if data == nil {
			progress = &TeamProgress{}
		} else {
			err := json.Unmarshal(data, &progress)
			if err != nil {
				return fmt.Errorf("db: unmarshal team progress for %q: %w", team, err)
			}
			exists = true
		}

		var err error
		if progress, err = modify(progress, exists); err != nil {
			return fmt.Errorf("db: modify team progress for %q: %w", team, err)
		}

		if progress == nil {
			if !exists {
				return nil
			}
			return b.Delete([]byte(team))
		}
		return b.Put([]byte(team), must(json.Marshal(progress)))
	})
}

func AddTeamProgress(team, puzzle string, time time.Time) error {
	return modify(team, func(progress *TeamProgress, _ bool) (*TeamProgress, error) {
		if progress.Puzzles == nil {
			progress.Puzzles = make(map[string]PuzzleProgress, 1)
		}

		if _, exists := progress.Puzzles[puzzle]; exists {
			return progress, nil
		}

		progress.Puzzles[puzzle] = PuzzleProgress{
			FirstOpened: time,
		}
		progress.Hidden = false

		return progress, nil
	})
}

func SetTeamHidden(team string, hidden bool) error {
	return modify(team, func(progress *TeamProgress, exists bool) (*TeamProgress, error) {
		if !exists {
			return nil, nil
		}

		progress.Hidden = hidden
		return progress, nil
	})
}

func RenameTeam(team, newName string) error {
	if db == nil {
		panic("db: not opened")
	}

	return db.Update(func(tx *bbolt.Tx) error {
		teamBucket := tx.Bucket(bucketTeams)
		if teamBucket == nil {
			return fmt.Errorf("db: teams bucket not found")
		}
		renamesBucket := tx.Bucket(bucketRenames)
		if renamesBucket == nil {
			return fmt.Errorf("db: renames bucket not found")
		}

		data := teamBucket.Get([]byte(team))
		if data == nil {
			return nil
		}

		if teamBucket.Get([]byte(newName)) != nil {
			return fmt.Errorf("db: team %s already exists", newName)
		}

		err := teamBucket.Put([]byte(newName), data)
		if err != nil {
			return err
		}

		err = teamBucket.Delete([]byte(team))
		if err != nil {
			return err
		}

		return renamesBucket.Put([]byte(team), []byte(newName))
	})
}

func TeamRename(team string) (string, error) {
	newName := team

	err := db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketRenames)
		if b == nil {
			return fmt.Errorf("db: renames bucket not found")
		}

		key := []byte(team)
		for newKey := b.Get(key); newKey != nil; newKey = b.Get(key) {
			key = newKey
		}
		newName = string(key)

		return nil
	})

	if err != nil {
		return "", err
	}

	return newName, nil
}

var errStop = fmt.Errorf("stop iteration")

func AllTeams() iter.Seq2[string, TeamProgress] {
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

func AllRenames() iter.Seq2[string, string] {
	if db == nil {
		panic("db: not opened")
	}

	return func(yield func(string, string) bool) {
		err := db.View(func(tx *bbolt.Tx) error {
			b := tx.Bucket(bucketRenames)
			if b == nil {
				return fmt.Errorf("db: renames bucket not found")
			}

			return b.ForEach(func(k, v []byte) error {
				if !yield(string(k), string(v)) {
					return errStop
				}
				return nil
			})
		})

		if err != nil {
			if errors.Is(err, errStop) {
				return
			}
			panic(fmt.Errorf("db: get all renames: %w", err))
		}
	}
}
