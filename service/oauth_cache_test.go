package service

import (
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/lib/cache"
)

func TestOauthCacheUsesSharedCacheAndDeletesState(t *testing.T) {
	Cache = cache.NewSimpleCache()
	service := &OauthService{}
	item := &OauthCacheItem{Op: "oidc", Action: "login", Nonce: "nonce"}
	service.SetOauthCache("state", item, 60)

	got := service.GetOauthCache("state")
	if got == nil || got.Nonce != item.Nonce {
		t.Fatalf("unexpected cached item: %#v", got)
	}

	service.DeleteOauthCache("state")
	if got := service.GetOauthCache("state"); got != nil {
		t.Fatalf("deleted OAuth state must not be reusable: %#v", got)
	}
}
