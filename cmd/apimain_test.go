package main

import (
	"strings"
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestVerifyDatabaseVersion(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open in-memory database: %v", err)
	}

	if err := VerifyDatabaseVersion(db, DatabaseVersion); err == nil || !strings.Contains(err.Error(), "versions table is missing") {
		t.Fatalf("missing version table must fail clearly, got %v", err)
	}
	if err := db.AutoMigrate(&model.Version{}); err != nil {
		t.Fatalf("create version table: %v", err)
	}
	if err := db.Create(&model.Version{Version: DatabaseVersion - 1}).Error; err != nil {
		t.Fatalf("insert old version: %v", err)
	}
	if err := VerifyDatabaseVersion(db, DatabaseVersion); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("mismatched version must fail clearly, got %v", err)
	}
	if err := db.Create(&model.Version{Version: DatabaseVersion}).Error; err != nil {
		t.Fatalf("insert expected version: %v", err)
	}
	if err := VerifyDatabaseVersion(db, DatabaseVersion); err != nil {
		t.Fatalf("expected database version to pass: %v", err)
	}
}
