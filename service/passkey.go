package service

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/lejianwen/rustdesk-api/v2/config"
	"github.com/lejianwen/rustdesk-api/v2/lib/cache"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/utils"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrPasskeyDisabled          = errors.New("PasskeyDisabled")
	ErrPasskeyChallengeExpired  = errors.New("PasskeyChallengeExpired")
	ErrPasskeyInvalidCredential = errors.New("PasskeyCredentialInvalid")
	ErrPasskeyCredentialRevoked = errors.New("PasskeyCredentialRevoked")
	ErrPasskeyCredentialExists  = errors.New("PasskeyCredentialExists")
)

const passkeyChallengeTTL = 5 * 60

type PasskeyService struct {
	server *webauthn.WebAuthn
}

type PasskeyChallengeItem struct {
	Kind      string               `json:"kind"`
	UserId    uint                 `json:"user_id"`
	RPID      string               `json:"rp_id"`
	Binding   string               `json:"binding,omitempty"`
	Session   webauthn.SessionData `json:"session"`
	ExpiresAt int64                `json:"expires_at"`
}

type PasskeyRegistrationBegin struct {
	Challenge string                       `json:"challenge"`
	PublicKey *protocol.CredentialCreation `json:"public_key"`
	ExpiresAt int64                        `json:"expires_at"`
}

type PasskeyLoginBegin struct {
	Challenge string                        `json:"challenge"`
	PublicKey *protocol.CredentialAssertion `json:"public_key"`
	ExpiresAt int64                         `json:"expires_at"`
}

// PasskeyLoginResult keeps the authenticated account and the exact
// credential that produced the session, allowing later device revocation to
// clean up sessions issued by that credential.
type PasskeyLoginResult struct {
	User         *model.User
	CredentialId uint
}

type passkeyUser struct {
	account     *model.User
	credentials []webauthn.Credential
	handle      []byte
}

func (u *passkeyUser) WebAuthnID() []byte { return u.handle }
func (u *passkeyUser) WebAuthnName() string {
	if u.account == nil {
		return "user-unknown"
	}
	if u.account.Username == "" {
		return fmt.Sprintf("user-%d", u.account.Id)
	}
	return u.account.Username
}
func (u *passkeyUser) WebAuthnDisplayName() string {
	if u.account == nil {
		return "Yinhe user"
	}
	if u.account.Nickname != "" {
		return u.account.Nickname
	}
	return u.WebAuthnName()
}
func (u *passkeyUser) WebAuthnCredentials() []webauthn.Credential { return u.credentials }

func NewPasskeyServiceFromConfig(c *config.Config) (*PasskeyService, error) {
	ps := &PasskeyService{}
	if c == nil || !c.Passkey.Enabled {
		return ps, nil
	}
	uv := protocol.VerificationPreferred
	if c.Passkey.RequireUserVerification {
		uv = protocol.VerificationRequired
	}
	server, err := webauthn.New(&webauthn.Config{
		RPID:          strings.TrimSpace(c.Passkey.RPID),
		RPDisplayName: strings.TrimSpace(c.Passkey.RPDisplayName),
		RPOrigins:     c.Passkey.Origins,
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			UserVerification: uv,
		},
		Timeouts: webauthn.TimeoutsConfig{
			Login:        webauthn.TimeoutConfig{Enforce: true, Timeout: 2 * time.Minute, TimeoutUVD: 2 * time.Minute},
			Registration: webauthn.TimeoutConfig{Enforce: true, Timeout: 2 * time.Minute, TimeoutUVD: 2 * time.Minute},
		},
	})
	if err != nil {
		return nil, err
	}
	ps.server = server
	return ps, nil
}

func (ps *PasskeyService) enabled() error {
	if ps == nil || ps.server == nil || Config == nil || !Config.Passkey.Enabled {
		return ErrPasskeyDisabled
	}
	return nil
}

func (ps *PasskeyService) beginCache(kind string, userID uint, binding string, session *webauthn.SessionData) (string, error) {
	if Cache == nil || session == nil {
		return "", ErrPasskeyChallengeExpired
	}
	challenge := utils.RandomString(48)
	if challenge == "" {
		return "", ErrPasskeyChallengeExpired
	}
	item := &PasskeyChallengeItem{
		Kind: kind, UserId: userID, RPID: session.RelyingPartyID, Binding: passkeyBinding(binding),
		Session: *session, ExpiresAt: time.Now().Add(passkeyChallengeTTL * time.Second).Unix(),
	}
	if err := Cache.Set("passkey:"+kind+":"+challenge, item, passkeyChallengeTTL); err != nil {
		return "", err
	}
	return challenge, nil
}

func (ps *PasskeyService) consumeCache(kind, challenge, binding string) (*PasskeyChallengeItem, error) {
	if Cache == nil || strings.TrimSpace(challenge) == "" {
		return nil, ErrPasskeyChallengeExpired
	}
	atomicCache, ok := Cache.(cache.AtomicHandler)
	if !ok {
		return nil, ErrPasskeyChallengeExpired
	}
	item := &PasskeyChallengeItem{}
	if err := atomicCache.GetAndDelete("passkey:"+kind+":"+strings.TrimSpace(challenge), item); err != nil || item.Session.Challenge == "" {
		return nil, ErrPasskeyChallengeExpired
	}
	if item.ExpiresAt < time.Now().Unix() || (!item.Session.Expires.IsZero() && item.Session.Expires.Before(time.Now())) {
		return nil, ErrPasskeyChallengeExpired
	}
	if item.Binding != "" && item.Binding != passkeyBinding(binding) {
		return nil, ErrPasskeyChallengeExpired
	}
	return item, nil
}

func passkeyBinding(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	digest := sha256.Sum256([]byte(value))
	return base64.RawURLEncoding.EncodeToString(digest[:])
}

func (ps *PasskeyService) ensureUserHandle(u *model.User) ([]byte, error) {
	if u == nil || u.Id == 0 {
		return nil, ErrPasskeyInvalidCredential
	}
	if u.WebAuthnUserHandle != "" {
		handle, err := base64.RawURLEncoding.DecodeString(u.WebAuthnUserHandle)
		if err == nil && len(handle) >= 16 && len(handle) <= 64 {
			return handle, nil
		}
	}
	var handle []byte
	err := DB.Transaction(func(tx *gorm.DB) error {
		locked := &model.User{}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", u.Id).First(locked).Error; err != nil {
			return err
		}
		if locked.WebAuthnUserHandle != "" {
			decoded, decodeErr := base64.RawURLEncoding.DecodeString(locked.WebAuthnUserHandle)
			if decodeErr == nil && len(decoded) >= 16 && len(decoded) <= 64 {
				handle = decoded
				u.WebAuthnUserHandle = locked.WebAuthnUserHandle
				return nil
			}
		}
		handle = make([]byte, 32)
		if _, err := rand.Read(handle); err != nil {
			return err
		}
		encoded := base64.RawURLEncoding.EncodeToString(handle)
		if err := tx.Model(locked).Update("web_authn_user_handle", encoded).Error; err != nil {
			return err
		}
		u.WebAuthnUserHandle = encoded
		return nil
	})
	if err != nil {
		return nil, err
	}
	return handle, nil
}

func (ps *PasskeyService) loadCredentials(userID uint, rpID string) ([]webauthn.Credential, error) {
	rows := make([]model.PasskeyCredential, 0)
	if err := DB.Where("user_id = ? AND rp_id = ? AND revoked_at IS NULL", userID, rpID).Order("id asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	credentials := make([]webauthn.Credential, 0, len(rows))
	for _, row := range rows {
		credential, err := restoreCredential(&row)
		if err != nil {
			return nil, err
		}
		credentials = append(credentials, *credential)
	}
	return credentials, nil
}

func restoreCredential(row *model.PasskeyCredential) (*webauthn.Credential, error) {
	if row == nil || row.CredentialData == "" || row.RevokedAt != nil {
		return nil, ErrPasskeyCredentialRevoked
	}
	credential := &webauthn.Credential{}
	if err := json.Unmarshal([]byte(row.CredentialData), credential); err != nil {
		return nil, err
	}
	credential.Flags = webauthn.NewCredentialFlags(protocol.AuthenticatorFlags(row.FlagsRaw))
	if len(credential.ID) == 0 || base64.RawURLEncoding.EncodeToString(credential.ID) != row.CredentialId {
		return nil, ErrPasskeyInvalidCredential
	}
	return credential, nil
}

func credentialRecord(u *model.User, rpID, name string, credential *webauthn.Credential) (*model.PasskeyCredential, error) {
	if u == nil || credential == nil || len(credential.ID) == 0 {
		return nil, ErrPasskeyInvalidCredential
	}
	data, err := json.Marshal(credential)
	if err != nil {
		return nil, err
	}
	handle, err := base64.RawURLEncoding.DecodeString(u.WebAuthnUserHandle)
	if err != nil {
		return nil, err
	}
	return &model.PasskeyCredential{
		UserId: u.Id, RPID: rpID,
		CredentialId: base64.RawURLEncoding.EncodeToString(credential.ID),
		UserHandle:   base64.RawURLEncoding.EncodeToString(handle),
		Name:         strings.TrimSpace(name), CredentialData: string(data),
		FlagsRaw:  uint8(credential.Flags.ProtocolValue()),
		SignCount: credential.Authenticator.SignCount, CloneWarning: credential.Authenticator.CloneWarning,
		BackupEligible: credential.Flags.BackupEligible, BackupState: credential.Flags.BackupState,
	}, nil
}

func (ps *PasskeyService) BeginRegistration(u *model.User, binding string) (*PasskeyRegistrationBegin, error) {
	if err := ps.enabled(); err != nil {
		return nil, err
	}
	if u == nil || u.Id == 0 || !AllService.UserService.CheckUserEnable(u) {
		return nil, ErrPasskeyInvalidCredential
	}
	adapter, err := ps.userAdapter(u)
	if err != nil {
		return nil, err
	}
	creation, session, err := ps.server.BeginRegistration(adapter,
		webauthn.WithResidentKeyRequirement(protocol.ResidentKeyRequirementRequired),
		webauthn.WithExclusions(passkeyCredentialDescriptors(adapter.credentials)),
	)
	if err != nil {
		return nil, err
	}
	challenge, err := ps.beginCache("register", u.Id, binding, session)
	if err != nil {
		return nil, err
	}
	return &PasskeyRegistrationBegin{Challenge: challenge, PublicKey: creation, ExpiresAt: time.Now().Add(passkeyChallengeTTL * time.Second).Unix()}, nil
}

func passkeyCredentialDescriptors(credentials []webauthn.Credential) []protocol.CredentialDescriptor {
	descriptors := make([]protocol.CredentialDescriptor, 0, len(credentials))
	for _, credential := range credentials {
		descriptors = append(descriptors, credential.Descriptor())
	}
	return descriptors
}

func (ps *PasskeyService) CompleteRegistration(challenge string, rawCredential []byte, name, binding string) (*model.PasskeyCredentialView, error) {
	if err := ps.enabled(); err != nil {
		return nil, err
	}
	item, err := ps.consumeCache("register", challenge, binding)
	if err != nil {
		return nil, err
	}
	u := AllService.UserService.InfoById(item.UserId)
	adapter, err := ps.userAdapter(u)
	if err != nil || !bytes.Equal(adapter.handle, item.Session.UserID) {
		return nil, ErrPasskeyInvalidCredential
	}
	credential, err := ps.finishRegistration(adapter, item.Session, rawCredential)
	if err != nil {
		return nil, ErrPasskeyInvalidCredential
	}
	record, err := credentialRecord(u, item.RPID, name, credential)
	if err != nil {
		return nil, err
	}
	var existing model.PasskeyCredential
	if err := DB.Where("rp_id = ? AND credential_id = ?", record.RPID, record.CredentialId).First(&existing).Error; err == nil && existing.Id > 0 {
		return nil, ErrPasskeyCredentialExists
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if err := DB.Create(record).Error; err != nil {
		return nil, err
	}
	return record.View(), nil
}

func (ps *PasskeyService) BeginLogin() (*PasskeyLoginBegin, error) {
	if err := ps.enabled(); err != nil {
		return nil, err
	}
	assertion, session, err := ps.server.BeginDiscoverableLogin()
	if err != nil {
		return nil, err
	}
	challenge, err := ps.beginCache("login", 0, "", session)
	if err != nil {
		return nil, err
	}
	return &PasskeyLoginBegin{Challenge: challenge, PublicKey: assertion, ExpiresAt: time.Now().Add(passkeyChallengeTTL * time.Second).Unix()}, nil
}

func (ps *PasskeyService) CompleteLogin(challenge string, rawCredential []byte) (*PasskeyLoginResult, error) {
	if err := ps.enabled(); err != nil {
		return nil, err
	}
	item, err := ps.consumeCache("login", challenge, "")
	if err != nil {
		return nil, err
	}
	var resolved *passkeyUser
	var resolvedCredentialID uint
	handler := func(rawID, userHandle []byte) (webauthn.User, error) {
		if len(rawID) == 0 || len(userHandle) == 0 {
			return nil, ErrPasskeyInvalidCredential
		}
		credentialID := base64.RawURLEncoding.EncodeToString(rawID)
		row := &model.PasskeyCredential{}
		if err := DB.Where("rp_id = ? AND credential_id = ? AND revoked_at IS NULL", item.RPID, credentialID).First(row).Error; err != nil {
			return nil, ErrPasskeyCredentialRevoked
		}
		u := AllService.UserService.InfoById(row.UserId)
		adapter, adapterErr := ps.userAdapter(u)
		if adapterErr != nil || !bytes.Equal(adapter.handle, userHandle) {
			return nil, ErrPasskeyInvalidCredential
		}
		resolved = adapter
		resolvedCredentialID = row.Id
		return adapter, nil
	}
	credential, err := ps.finishPasskeyLogin(handler, item.Session, rawCredential)
	if err != nil || resolved == nil || credential == nil {
		return nil, ErrPasskeyInvalidCredential
	}
	if err := ps.updateAfterLogin(resolved.account.Id, item.RPID, credential); err != nil {
		return nil, err
	}
	return &PasskeyLoginResult{User: resolved.account, CredentialId: resolvedCredentialID}, nil
}

func (ps *PasskeyService) updateAfterLogin(userID uint, rpID string, credential *webauthn.Credential) error {
	credentialID := base64.RawURLEncoding.EncodeToString(credential.ID)
	data, err := json.Marshal(credential)
	if err != nil {
		return err
	}
	now := time.Now().Unix()
	return DB.Transaction(func(tx *gorm.DB) error {
		row := &model.PasskeyCredential{}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ? AND rp_id = ? AND credential_id = ? AND revoked_at IS NULL", userID, rpID, credentialID).First(row).Error; err != nil {
			return ErrPasskeyCredentialRevoked
		}
		return tx.Model(row).Updates(map[string]interface{}{
			"credential_data": string(data),
			"flags_raw":       uint8(credential.Flags.ProtocolValue()),
			"sign_count":      credential.Authenticator.SignCount,
			"clone_warning":   credential.Authenticator.CloneWarning,
			"backup_eligible": credential.Flags.BackupEligible,
			"backup_state":    credential.Flags.BackupState,
			"last_used_at":    now,
		}).Error
	})
}

func (ps *PasskeyService) userAdapter(u *model.User) (*passkeyUser, error) {
	if u == nil || u.Id == 0 || !AllService.UserService.CheckUserEnable(u) {
		return nil, ErrPasskeyInvalidCredential
	}
	handle, err := ps.ensureUserHandle(u)
	if err != nil {
		return nil, err
	}
	credentials, err := ps.loadCredentials(u.Id, ps.server.Config.RPID)
	if err != nil {
		return nil, err
	}
	return &passkeyUser{account: u, credentials: credentials, handle: handle}, nil
}

func requestFromCredential(raw []byte) (*http.Request, error) {
	if len(raw) == 0 || !json.Valid(raw) {
		return nil, ErrPasskeyInvalidCredential
	}
	req, err := http.NewRequest(http.MethodPost, "http://passkey.invalid/ceremony", io.NopCloser(bytes.NewReader(raw)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func (ps *PasskeyService) finishRegistration(user webauthn.User, session webauthn.SessionData, raw []byte) (*webauthn.Credential, error) {
	req, err := requestFromCredential(raw)
	if err != nil {
		return nil, err
	}
	return ps.server.FinishRegistration(user, session, req)
}

func (ps *PasskeyService) finishPasskeyLogin(handler webauthn.DiscoverableUserHandler, session webauthn.SessionData, raw []byte) (*webauthn.Credential, error) {
	req, err := requestFromCredential(raw)
	if err != nil {
		return nil, err
	}
	return ps.server.FinishDiscoverableLogin(handler, session, req)
}

func (ps *PasskeyService) List(u *model.User, includeRevoked bool) ([]*model.PasskeyCredentialView, error) {
	if err := ps.enabled(); err != nil {
		return nil, err
	}
	if u == nil || u.Id == 0 {
		return nil, ErrPasskeyInvalidCredential
	}
	query := DB.Where("user_id = ? AND rp_id = ?", u.Id, ps.server.Config.RPID).Order("id asc")
	if !includeRevoked {
		query = query.Where("revoked_at IS NULL")
	}
	rows := make([]model.PasskeyCredential, 0)
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]*model.PasskeyCredentialView, 0, len(rows))
	for i := range rows {
		result = append(result, rows[i].View())
	}
	return result, nil
}

func (ps *PasskeyService) Revoke(actor *model.User, credentialID uint, reason string) (int64, error) {
	if err := ps.enabled(); err != nil {
		return 0, err
	}
	if actor == nil || actor.Id == 0 || credentialID == 0 {
		return 0, ErrPasskeyInvalidCredential
	}
	return ps.revokeCredential(actor.Id, actor.Id, credentialID, reason)
}

// RevokeForAdmin is intentionally separate from Revoke so a caller cannot
// accidentally bypass the owner/admin authorization boundary.
func (ps *PasskeyService) RevokeForAdmin(actor *model.User, userID, credentialID uint, reason string) (int64, error) {
	if actor == nil || !isPasskeyAdmin(actor) {
		return 0, ErrPasskeyInvalidCredential
	}
	if err := ps.enabled(); err != nil {
		return 0, err
	}
	return ps.revokeCredential(userID, actor.Id, credentialID, reason)
}

// revokeCredential is transactional: the credential is soft-revoked and all
// sessions issued by it are removed in the same database transaction. This
// prevents a revoked authenticator from leaving an active bearer session
// behind.
func (ps *PasskeyService) revokeCredential(userID, actorID, credentialID uint, reason string) (int64, error) {
	row := &model.PasskeyCredential{}
	var sessionsRevoked int64
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND user_id = ? AND rp_id = ? AND revoked_at IS NULL", credentialID, userID, ps.server.Config.RPID).First(row).Error; err != nil {
			return ErrPasskeyCredentialRevoked
		}
		now := time.Now().Unix()
		if err := tx.Model(row).Where("revoked_at IS NULL").Updates(map[string]interface{}{
			"revoked_at":        &now,
			"revoked_by":        actorID,
			"revocation_reason": strings.TrimSpace(reason),
		}).Error; err != nil {
			return err
		}
		result := tx.Where("user_id = ? AND passkey_credential_id = ?", userID, row.Id).Delete(&model.UserToken{})
		sessionsRevoked = result.RowsAffected
		return result.Error
	})
	return sessionsRevoked, err
}

func isPasskeyAdmin(u *model.User) bool {
	return u != nil && (u.Role == model.RoleAdmin || u.Role == model.RoleOwner || (u.IsAdmin != nil && *u.IsAdmin))
}
