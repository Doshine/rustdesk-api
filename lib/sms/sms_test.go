package sms

import (
	"testing"

	log "github.com/sirupsen/logrus"
)

func TestNewSenderDoesNotFallbackToMock(t *testing.T) {
	for _, cfg := range []*Config{nil, {}, {Provider: "unknown"}, {Provider: ProviderAliyun}} {
		if sender, err := NewSender(cfg, log.New()); err == nil || sender != nil {
			t.Fatalf("invalid config must fail closed: sender=%T err=%v", sender, err)
		}
	}
}

func TestMockSenderRequiresExplicitProvider(t *testing.T) {
	sender, err := NewSender(&Config{Provider: ProviderMock}, log.New())
	if err != nil {
		t.Fatalf("explicit development mock should initialize: %v", err)
	}
	if _, ok := sender.(*MockSender); !ok {
		t.Fatalf("expected mock sender, got %T", sender)
	}
}
