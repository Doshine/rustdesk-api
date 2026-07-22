package service

import (
	"errors"
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/model"
)

func TestAuditServiceIsAppendOnly(t *testing.T) {
	as := &AuditService{}

	checks := []struct {
		name string
		call func() error
	}{
		{name: "delete connection", call: func() error { return as.DeleteAuditConn(&model.AuditConn{}) }},
		{name: "update connection", call: func() error { return as.UpdateAuditConn(&model.AuditConn{}) }},
		{name: "batch delete connection", call: func() error { return as.BatchDeleteAuditConn([]uint{1}) }},
		{name: "delete file", call: func() error { return as.DeleteAuditFile(&model.AuditFile{}) }},
		{name: "update file", call: func() error { return as.UpdateAuditFile(&model.AuditFile{}) }},
		{name: "batch delete file", call: func() error { return as.BatchDeleteAuditFile([]uint{1}) }},
	}

	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			if !errors.Is(check.call(), model.ErrAuditAppendOnly) {
				t.Fatalf("expected append-only error")
			}
		})
	}
}
