package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1" // RFC 6238 TOTP uses HMAC-SHA1 for broad authenticator compatibility.
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/lib/cache"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/utils"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrMfaDisabled             = errors.New("MfaDisabled")
	ErrMfaNotEnrolled          = errors.New("MfaNotEnrolled")
	ErrMfaCodeInvalid          = errors.New("MfaCodeError")
	ErrMfaAlreadyEnabled       = errors.New("MfaAlreadyEnabled")
	ErrMfaRequired             = errors.New("MfaEnrollmentRequired")
	ErrMfaStepUpInvalid        = errors.New("MfaStepUpError")
	ErrMfaChallengeExpired     = errors.New("MfaChallengeExpired")
	ErrMfaChallengeCodeInvalid = errors.New("MfaChallengeCodeError")
	ErrMfaEnrollmentExpired    = errors.New("MfaEnrollmentChallengeExpired")
)

type MfaService struct{}

type MfaEnrollment struct {
	Secret      string   `json:"secret"`
	OtpAuthUrl  string   `json:"otpauth_url"`
	BackupCodes []string `json:"backup_codes"`
}

type MfaChallengeItem struct {
	UserId              uint   `json:"user_id"`
	Id                  string `json:"id"`
	Uuid                string `json:"uuid"`
	DeviceOs            string `json:"device_os"`
	DeviceType          string `json:"device_type"`
	ExpiresAt           int64  `json:"expires_at"`
	Attempts            int    `json:"attempts"`
	LoginType           string `json:"login_type,omitempty"`
	PasskeyCredentialId uint   `json:"passkey_credential_id,omitempty"`
}

type MfaEnrollmentChallengeItem struct {
	UserId              uint   `json:"user_id"`
	Id                  string `json:"id"`
	Uuid                string `json:"uuid"`
	DeviceOs            string `json:"device_os"`
	DeviceType          string `json:"device_type"`
	LoginType           string `json:"login_type"`
	ExpiresAt           int64  `json:"expires_at"`
	Attempts            int    `json:"attempts"`
	Started             bool   `json:"started"`
	PasskeyCredentialId uint   `json:"passkey_credential_id,omitempty"`
}

const (
	mfaChallengeTTL      = 5 * 60
	mfaChallengeMaxTries = 5
	mfaEnrollmentTTL     = 10 * 60
)

func (ms *MfaService) Enroll(u *model.User) (*MfaEnrollment, error) {
	if !Config.Mfa.Enabled {
		return nil, ErrMfaDisabled
	}
	if u == nil || u.Id == 0 {
		return nil, errors.New("user is required")
	}
	if u.MfaEnabled {
		return nil, ErrMfaAlreadyEnabled
	}
	secret, err := randomSecret()
	if err != nil {
		return nil, err
	}
	backupCodes, err := randomBackupCodes(8)
	if err != nil {
		return nil, err
	}
	encryptedSecret, err := encryptMfaValue(secret)
	if err != nil {
		return nil, err
	}
	encryptedBackupCodes, err := encryptMfaValue(strings.Join(backupCodes, "\n"))
	if err != nil {
		return nil, err
	}
	if err := DB.Model(&model.User{}).Where("id = ?", u.Id).Updates(map[string]interface{}{
		"mfa_enabled":                false,
		"mfa_secret_encrypted":       encryptedSecret,
		"mfa_backup_codes_encrypted": encryptedBackupCodes,
	}).Error; err != nil {
		return nil, err
	}
	u.MfaEnabled = false
	u.MfaSecretEncrypted = encryptedSecret
	u.MfaBackupCodesEncrypted = encryptedBackupCodes
	return newMfaEnrollment(u, secret, backupCodes), nil
}

func newMfaEnrollment(u *model.User, secret string, backupCodes []string) *MfaEnrollment {
	issuer := Config.Mfa.Issuer
	if issuer == "" {
		issuer = "Yinhe"
	}
	label := url.QueryEscape(issuer + ":" + u.Username)
	otpauthURL := fmt.Sprintf("otpauth://totp/%s?secret=%s&issuer=%s", label, secret, url.QueryEscape(issuer))
	return &MfaEnrollment{Secret: secret, OtpAuthUrl: otpauthURL, BackupCodes: backupCodes}
}

func storedMfaEnrollment(u *model.User) (*MfaEnrollment, error) {
	if u == nil || u.MfaSecretEncrypted == "" || u.MfaBackupCodesEncrypted == "" {
		return nil, ErrMfaNotEnrolled
	}
	secret, err := decryptMfaValue(u.MfaSecretEncrypted)
	if err != nil || secret == "" {
		return nil, ErrMfaNotEnrolled
	}
	backupCodes, err := decryptMfaValue(u.MfaBackupCodesEncrypted)
	if err != nil {
		return nil, ErrMfaNotEnrolled
	}
	codes := make([]string, 0, 8)
	for _, code := range strings.Split(backupCodes, "\n") {
		if strings.TrimSpace(code) != "" {
			codes = append(codes, code)
		}
	}
	return newMfaEnrollment(u, secret, codes), nil
}

func (ms *MfaService) Enable(u *model.User, code string) error {
	if u == nil || u.Id == 0 {
		return errors.New("user is required")
	}
	if u.MfaEnabled {
		return ErrMfaAlreadyEnabled
	}
	secret, err := decryptMfaValue(u.MfaSecretEncrypted)
	if err != nil || secret == "" {
		return ErrMfaNotEnrolled
	}
	if !VerifyTotp(secret, code, time.Now()) {
		return ErrMfaCodeInvalid
	}
	if err := DB.Model(&model.User{}).Where("id = ?", u.Id).Update("mfa_enabled", true).Error; err != nil {
		return err
	}
	u.MfaEnabled = true
	return nil
}

func (ms *MfaService) Disable(u *model.User, code, password string) error {
	if u == nil || u.Id == 0 {
		return errors.New("user is required")
	}
	if !u.MfaEnabled {
		return ErrMfaNotEnrolled
	}
	if !AllService.UserService.VerifyCurrentPassword(u, password) {
		return ErrMfaStepUpInvalid
	}
	if !ms.VerifyUserCode(u, code) {
		return ErrMfaCodeInvalid
	}
	if err := DB.Model(&model.User{}).Where("id = ?", u.Id).Updates(map[string]interface{}{
		"mfa_enabled":                false,
		"mfa_secret_encrypted":       "",
		"mfa_backup_codes_encrypted": "",
	}).Error; err != nil {
		return err
	}
	u.MfaEnabled = false
	u.MfaSecretEncrypted = ""
	u.MfaBackupCodesEncrypted = ""
	return nil
}

// VerifyUserCode accepts either the current TOTP or one unused backup code.
func (ms *MfaService) VerifyUserCode(u *model.User, code string) bool {
	if u == nil || !u.MfaEnabled {
		return true
	}
	secret, err := decryptMfaValue(u.MfaSecretEncrypted)
	if err == nil && VerifyTotp(secret, code, time.Now()) {
		return true
	}
	return ms.consumeBackupCode(u.Id, code)
}

func (ms *MfaService) VerifyLogin(u *model.User, code string) error {
	if u == nil {
		return ErrMfaCodeInvalid
	}
	if !ms.RequiresMfaForLogin(u) {
		return nil
	}
	if !u.MfaEnabled {
		return ErrMfaRequired
	}
	if !ms.VerifyUserCode(u, code) {
		return ErrMfaCodeInvalid
	}
	return nil
}

func (ms *MfaService) RequiresMfaForLogin(u *model.User) bool {
	if u == nil {
		return false
	}
	return u.MfaEnabled || (Config.Mfa.RequiredForAdmin && (u.Role == model.RoleAdmin || u.Role == model.RoleOwner || (u.IsAdmin != nil && *u.IsAdmin)))
}

func (ms *MfaService) CreateEnrollmentChallenge(u *model.User, source *MfaEnrollmentChallengeItem) (string, error) {
	if !Config.Mfa.Enabled {
		return "", ErrMfaDisabled
	}
	if u == nil || u.Id == 0 || u.MfaEnabled || source == nil || Cache == nil {
		return "", ErrMfaEnrollmentExpired
	}
	challenge := utils.RandomString(48)
	if challenge == "" {
		return "", ErrMfaEnrollmentExpired
	}
	item := &MfaEnrollmentChallengeItem{
		UserId:              u.Id,
		Id:                  source.Id,
		Uuid:                source.Uuid,
		DeviceOs:            source.DeviceOs,
		DeviceType:          source.DeviceType,
		LoginType:           source.LoginType,
		PasskeyCredentialId: source.PasskeyCredentialId,
		ExpiresAt:           time.Now().Add(mfaEnrollmentTTL * time.Second).Unix(),
	}
	if err := Cache.Set("mfa:enroll:"+challenge, item, mfaEnrollmentTTL); err != nil {
		return "", err
	}
	return challenge, nil
}

func (ms *MfaService) BeginEnrollmentChallenge(challenge string) (*model.User, *MfaEnrollment, error) {
	if Cache == nil || strings.TrimSpace(challenge) == "" {
		return nil, nil, ErrMfaEnrollmentExpired
	}
	challenge = strings.TrimSpace(challenge)
	key := "mfa:enroll:" + challenge
	item := &MfaEnrollmentChallengeItem{}
	if err := Cache.Get(key, item); err != nil || item.UserId == 0 || item.ExpiresAt <= time.Now().Unix() {
		return nil, nil, ErrMfaEnrollmentExpired
	}
	u := AllService.UserService.InfoById(item.UserId)
	if u == nil || u.Id == 0 || !AllService.UserService.CheckUserEnable(u) || u.MfaEnabled {
		return nil, nil, ErrMfaEnrollmentExpired
	}
	if !item.Started {
		enrollment, err := storedMfaEnrollment(u)
		if err != nil {
			enrollment, err = ms.Enroll(u)
		}
		if err != nil {
			return nil, nil, err
		}
		item.Started = true
		expiresIn := int(item.ExpiresAt - time.Now().Unix())
		if expiresIn < 1 {
			return nil, nil, ErrMfaEnrollmentExpired
		}
		if err := Cache.Set(key, item, expiresIn); err != nil {
			return nil, nil, err
		}
		return u, enrollment, nil
	}
	enrollment, err := storedMfaEnrollment(u)
	if err != nil {
		return nil, nil, ErrMfaEnrollmentExpired
	}
	return u, enrollment, nil
}

func (ms *MfaService) CompleteEnrollmentChallenge(challenge, code, ip string) (*model.User, *model.UserToken, error) {
	if !Config.Mfa.Enabled {
		return nil, nil, ErrMfaDisabled
	}
	if Cache == nil || strings.TrimSpace(challenge) == "" {
		return nil, nil, ErrMfaEnrollmentExpired
	}
	challenge = strings.TrimSpace(challenge)
	key := "mfa:enroll:" + challenge
	attemptsKey := key + ":attempts"
	item := &MfaEnrollmentChallengeItem{}
	if err := Cache.Get(key, item); err != nil || item.UserId == 0 || !item.Started || item.ExpiresAt <= time.Now().Unix() {
		return nil, nil, ErrMfaEnrollmentExpired
	}
	u := AllService.UserService.InfoById(item.UserId)
	if u == nil || u.Id == 0 || !AllService.UserService.CheckUserEnable(u) || u.MfaEnabled {
		return nil, nil, ErrMfaEnrollmentExpired
	}
	secret, err := decryptMfaValue(u.MfaSecretEncrypted)
	if err != nil || !VerifyTotp(secret, code, time.Now()) {
		attempts := item.Attempts + 1
		if atomic, ok := Cache.(cache.AtomicHandler); ok {
			expiresIn := int(item.ExpiresAt - time.Now().Unix())
			if expiresIn < 1 {
				expiresIn = 1
			}
			count, countErr := atomic.Increment(attemptsKey, expiresIn)
			if countErr != nil {
				return nil, nil, ErrMfaEnrollmentExpired
			}
			attempts = int(count)
		}
		item.Attempts = attempts
		if item.Attempts >= mfaChallengeMaxTries {
			_ = Cache.Delete(key)
			_ = Cache.Delete(attemptsKey)
			return nil, nil, ErrMfaEnrollmentExpired
		}
		if _, ok := Cache.(cache.AtomicHandler); !ok {
			expiresIn := int(item.ExpiresAt - time.Now().Unix())
			if expiresIn < 1 {
				expiresIn = 1
			}
			_ = Cache.Set(key, item, expiresIn)
		}
		return nil, nil, ErrMfaChallengeCodeInvalid
	}
	if atomic, ok := Cache.(cache.AtomicHandler); ok {
		consumed := &MfaEnrollmentChallengeItem{}
		if err := atomic.GetAndDelete(key, consumed); err != nil || consumed.UserId == 0 {
			return nil, nil, ErrMfaEnrollmentExpired
		}
		_ = Cache.Delete(attemptsKey)
	} else if err := Cache.Delete(key); err != nil {
		return nil, nil, ErrMfaEnrollmentExpired
	}
	if err := DB.Model(&model.User{}).Where("id = ?", u.Id).Updates(map[string]interface{}{
		"mfa_enabled": true,
	}).Error; err != nil {
		return nil, nil, err
	}
	u.MfaEnabled = true
	loginType := item.LoginType
	if loginType == "" {
		loginType = model.LoginLogTypeAccount
	}
	loginLog := &model.LoginLog{
		UserId:              u.Id,
		Client:              item.DeviceType,
		DeviceId:            item.Id,
		Uuid:                item.Uuid,
		Ip:                  ip,
		Type:                loginType,
		Platform:            item.DeviceOs,
		PasskeyCredentialId: item.PasskeyCredentialId,
	}
	var ut *model.UserToken
	var loginErr error
	if item.PasskeyCredentialId != 0 {
		ut, loginErr = AllService.UserService.LoginWithPasskey(u, loginLog)
	} else {
		ut = AllService.UserService.Login(u, loginLog)
	}
	if ut == nil {
		if loginErr != nil {
			return nil, nil, loginErr
		}
		return nil, nil, errors.New("LoginFailed")
	}
	return u, ut, nil
}

func (ms *MfaService) CreateOauthChallenge(u *model.User, source *OauthCacheItem) (string, error) {
	if u == nil || u.Id == 0 || source == nil || !u.MfaEnabled || Cache == nil {
		return "", ErrMfaChallengeExpired
	}
	challenge := utils.RandomString(48)
	if challenge == "" {
		return "", ErrMfaChallengeExpired
	}
	item := &MfaChallengeItem{
		UserId:              u.Id,
		Id:                  source.Id,
		Uuid:                source.Uuid,
		DeviceOs:            source.DeviceOs,
		DeviceType:          source.DeviceType,
		LoginType:           source.LoginType,
		PasskeyCredentialId: source.PasskeyCredentialId,
		ExpiresAt:           time.Now().Add(mfaChallengeTTL * time.Second).Unix(),
	}
	if err := Cache.Set("mfa:oauth:"+challenge, item, mfaChallengeTTL); err != nil {
		return "", err
	}
	return challenge, nil
}

func (ms *MfaService) CompleteOauthChallenge(challenge, code, ip string) (*model.User, *model.UserToken, error) {
	if Cache == nil || strings.TrimSpace(challenge) == "" {
		return nil, nil, ErrMfaChallengeExpired
	}
	challenge = strings.TrimSpace(challenge)
	key := "mfa:oauth:" + challenge
	attemptsKey := key + ":attempts"
	item := &MfaChallengeItem{}
	if err := Cache.Get(key, item); err != nil || item.UserId == 0 || item.ExpiresAt <= time.Now().Unix() {
		return nil, nil, ErrMfaChallengeExpired
	}
	u := AllService.UserService.InfoById(item.UserId)
	if u == nil || u.Id == 0 || !u.MfaEnabled {
		return nil, nil, ErrMfaChallengeExpired
	}
	if !ms.VerifyUserCode(u, code) {
		attempts := item.Attempts + 1
		if atomic, ok := Cache.(cache.AtomicHandler); ok {
			expiresIn := int(item.ExpiresAt - time.Now().Unix())
			if expiresIn < 1 {
				expiresIn = 1
			}
			count, err := atomic.Increment(attemptsKey, expiresIn)
			if err != nil {
				return nil, nil, ErrMfaChallengeExpired
			}
			attempts = int(count)
		}
		item.Attempts = attempts
		if item.Attempts >= mfaChallengeMaxTries {
			_ = Cache.Delete(key)
			_ = Cache.Delete(attemptsKey)
			return nil, nil, ErrMfaChallengeExpired
		}
		if _, ok := Cache.(cache.AtomicHandler); !ok {
			expiresIn := int(item.ExpiresAt - time.Now().Unix())
			if expiresIn < 1 {
				expiresIn = 1
			}
			_ = Cache.Set(key, item, expiresIn)
		}
		return nil, nil, ErrMfaChallengeCodeInvalid
	}

	// Redis production caches implement GetAndDelete, making successful
	// challenge consumption one-time even when two requests race. Development
	// caches fall back to Delete and remain protected by short expiry/attempts.
	if atomic, ok := Cache.(cache.AtomicHandler); ok {
		consumed := &MfaChallengeItem{}
		if err := atomic.GetAndDelete(key, consumed); err != nil || consumed.UserId == 0 {
			return nil, nil, ErrMfaChallengeExpired
		}
		_ = Cache.Delete(attemptsKey)
	} else if err := Cache.Delete(key); err != nil {
		return nil, nil, ErrMfaChallengeExpired
	}
	loginType := item.LoginType
	if loginType == "" {
		loginType = model.LoginLogTypeOauth
	}
	loginLog := &model.LoginLog{
		UserId:              u.Id,
		Client:              item.DeviceType,
		DeviceId:            item.Id,
		Uuid:                item.Uuid,
		Ip:                  ip,
		Type:                loginType,
		Platform:            item.DeviceOs,
		PasskeyCredentialId: item.PasskeyCredentialId,
	}
	var ut *model.UserToken
	var loginErr error
	if item.PasskeyCredentialId != 0 {
		ut, loginErr = AllService.UserService.LoginWithPasskey(u, loginLog)
	} else {
		ut = AllService.UserService.Login(u, loginLog)
	}
	if ut == nil {
		if loginErr != nil {
			return nil, nil, loginErr
		}
		return nil, nil, errors.New("LoginFailed")
	}
	return u, ut, nil
}

func (ms *MfaService) consumeBackupCode(userID uint, code string) bool {
	if userID == 0 || strings.TrimSpace(code) == "" {
		return false
	}
	consumed := false
	_ = DB.Transaction(func(tx *gorm.DB) error {
		var user model.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, userID).Error; err != nil {
			return err
		}
		decoded, err := decryptMfaValue(user.MfaBackupCodesEncrypted)
		if err != nil {
			return err
		}
		candidate := normalizeBackupCode(code)
		codes := strings.Split(decoded, "\n")
		remaining := make([]string, 0, len(codes))
		for _, item := range codes {
			if item != "" && subtle.ConstantTimeCompare([]byte(normalizeBackupCode(item)), []byte(candidate)) == 1 && !consumed {
				consumed = true
				continue
			}
			if item != "" {
				remaining = append(remaining, item)
			}
		}
		if !consumed {
			return nil
		}
		next, err := encryptMfaValue(strings.Join(remaining, "\n"))
		if err != nil {
			return err
		}
		return tx.Model(&model.User{}).Where("id = ?", userID).Update("mfa_backup_codes_encrypted", next).Error
	})
	return consumed
}

func randomSecret() (string, error) {
	b := make([]byte, 20)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b), nil
}

func randomBackupCodes(count int) ([]string, error) {
	codes := make([]string, 0, count)
	for i := 0; i < count; i++ {
		b := make([]byte, 5)
		if _, err := rand.Read(b); err != nil {
			return nil, err
		}
		codes = append(codes, strings.ToUpper(hex.EncodeToString(b)))
	}
	return codes, nil
}

func VerifyTotp(secret, code string, now time.Time) bool {
	code = strings.TrimSpace(code)
	if len(code) != 6 {
		return false
	}
	for offset := -1; offset <= 1; offset++ {
		candidate := totpCode(secret, now.Add(time.Duration(offset)*30*time.Second))
		if subtle.ConstantTimeCompare([]byte(candidate), []byte(code)) == 1 {
			return true
		}
	}
	return false
}

func totpCode(secret string, now time.Time) string {
	decoded, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(strings.TrimSpace(secret)))
	if err != nil || len(decoded) == 0 {
		return ""
	}
	counter := uint64(now.Unix() / 30)
	var message [8]byte
	binary.BigEndian.PutUint64(message[:], counter)
	h := hmac.New(sha1.New, decoded)
	_, _ = h.Write(message[:])
	sum := h.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	value := (uint32(sum[offset])&0x7f)<<24 |
		(uint32(sum[offset+1]) << 16) |
		(uint32(sum[offset+2]) << 8) |
		uint32(sum[offset+3])
	return fmt.Sprintf("%06d", value%1000000)
}

func encryptMfaValue(value string) (string, error) {
	block, err := mfaCipher()
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, []byte(value), nil)
	return base64.RawStdEncoding.EncodeToString(sealed), nil
}

func decryptMfaValue(value string) (string, error) {
	if value == "" {
		return "", ErrMfaNotEnrolled
	}
	block, err := mfaCipher()
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	sealed, err := base64.RawStdEncoding.DecodeString(value)
	if err != nil || len(sealed) < gcm.NonceSize() {
		return "", ErrMfaNotEnrolled
	}
	nonce, ciphertext := sealed[:gcm.NonceSize()], sealed[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", ErrMfaNotEnrolled
	}
	return string(plaintext), nil
}

func mfaCipher() (cipher.Block, error) {
	keyMaterial := Config.Mfa.EncryptionKey
	if keyMaterial == "" {
		keyMaterial = Config.Jwt.Key
	}
	if len(keyMaterial) < 32 {
		return nil, errors.New("MFA encryption key must contain at least 32 characters")
	}
	key := sha256.Sum256([]byte(keyMaterial))
	return aes.NewCipher(key[:])
}

func normalizeBackupCode(code string) string {
	return strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(code), "-", ""), " ", ""))
}
