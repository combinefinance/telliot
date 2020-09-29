// Copyright (c) The Tellor Authors.
// Licensed under the MIT License.

package db

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDB(t *testing.T) {
	db, err := Open(filepath.Join(os.TempDir(), "test_db"))
	defer func() {
		if err := db.Close(); err != nil {
			t.Error(err)
		}
	}()

	if err != nil {
		t.Error(err)
	}
	err = db.Put("sample", []byte("sample_value"))
	if err != nil {
		t.Error(err)
	}
	b, err := db.Has("sample")
	if err != nil {
		t.Error(err)
	}
	if !b {
		t.Error("Expected key to be present in DB")
	}
	v, err := db.Get("sample")
	if err != nil {
		t.Error(err)
	}
	s := string(v)
	if s != "sample_value" {
		t.Error("Get value doesn't match original: " + s + " != sample_value")
	}
	t.Log("Retrieved " + s)
}
