package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrDeploymentCodeInvalid   = errors.New("deployment code is invalid")
	ErrDeploymentCodeExpired   = errors.New("deployment code has expired")
	ErrDeploymentCodeRevoked   = errors.New("deployment code has been revoked")
	ErrDeploymentCodeExhausted = errors.New("deployment code usage limit reached")
	ErrDeploymentPlatform      = errors.New("platform is not allowed")
	ErrDeviceAlreadyEnrolled   = errors.New("device is already enrolled")
)

var validDeploymentPlatforms = map[string]struct{}{
	"windows": {}, "macos": {}, "linux": {}, "android": {}, "ios": {},
}

type DeploymentCodeService struct{}

type DeploymentCodeCreateInput struct {
	Name             string
	DeviceGroupId    uint
	AllowedPlatforms []string
	ExpiresAt        int64
	MaxUses          int
	RotatedFromId    uint
}

type DeploymentCodeSecret struct {
	DeploymentCode *model.DeploymentCode `json:"deployment_code"`
	Code           string                `json:"code"`
}

type DeploymentClaimInput struct {
	Code     string
	DeviceId string
	Uuid     string
	Hostname string
	Os       string
	Username string
	Version  string
	Platform string
	Ip       string
}

func NormalizeDeploymentPlatform(platform string) string {
	p := strings.ToLower(strings.TrimSpace(platform))
	switch {
	case strings.Contains(p, "win"):
		return "windows"
	case strings.Contains(p, "darwin"), strings.Contains(p, "mac"), strings.Contains(p, "os x"):
		return "macos"
	case strings.Contains(p, "android"):
		return "android"
	case p == "ios", strings.Contains(p, "iphone"), strings.Contains(p, "ipad"):
		return "ios"
	case strings.Contains(p, "linux"):
		return "linux"
	default:
		return p
	}
}

func normalizeDeploymentPlatforms(platforms []string) ([]string, error) {
	seen := make(map[string]struct{})
	result := make([]string, 0, len(platforms))
	for _, value := range platforms {
		platform := NormalizeDeploymentPlatform(value)
		if _, ok := validDeploymentPlatforms[platform]; !ok {
			return nil, ErrDeploymentPlatform
		}
		if _, ok := seen[platform]; !ok {
			seen[platform] = struct{}{}
			result = append(result, platform)
		}
	}
	if len(result) == 0 {
		return nil, ErrDeploymentPlatform
	}
	return result, nil
}

func normalizeDeploymentCode(code string) string {
	replacer := strings.NewReplacer("-", "", " ", "", "\t", "", "\n", "")
	return strings.ToUpper(replacer.Replace(strings.TrimSpace(code)))
}

func deploymentCodeHash(code string) string {
	sum := sha256.Sum256([]byte(normalizeDeploymentCode(code)))
	return hex.EncodeToString(sum[:])
}

func generateDeploymentCode() (string, error) {
	random := make([]byte, 10)
	if _, err := rand.Read(random); err != nil {
		return "", fmt.Errorf("generate deployment code: %w", err)
	}
	raw := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(random)
	return fmt.Sprintf("YJ-%s-%s-%s-%s", raw[0:4], raw[4:8], raw[8:12], raw[12:16]), nil
}

func deploymentCodeHint(code string) string {
	parts := strings.Split(code, "-")
	if len(parts) != 5 {
		return "YJ-••••-••••-••••"
	}
	return fmt.Sprintf("%s-%s-••••-%s", parts[0], parts[1], parts[4])
}

func (s *DeploymentCodeService) Create(input DeploymentCodeCreateInput, actorUserId uint, ip string) (*DeploymentCodeSecret, error) {
	var result *DeploymentCodeSecret
	err := DB.Transaction(func(tx *gorm.DB) error {
		created, err := s.createWithTx(tx, input, actorUserId, ip, model.DeploymentAuditCreated)
		result = created
		return err
	})
	return result, err
}

func (s *DeploymentCodeService) createWithTx(tx *gorm.DB, input DeploymentCodeCreateInput, actorUserId uint, ip, action string) (*DeploymentCodeSecret, error) {
	input.Name = strings.TrimSpace(input.Name)
	now := time.Now().Unix()
	if input.Name == "" || len(input.Name) > 80 || input.MaxUses < 1 || input.MaxUses > 10000 ||
		input.ExpiresAt <= now || input.ExpiresAt > now+90*24*60*60 {
		return nil, ErrDeploymentCodeInvalid
	}
	platforms, err := normalizeDeploymentPlatforms(input.AllowedPlatforms)
	if err != nil {
		return nil, err
	}
	if input.DeviceGroupId > 0 {
		var count int64
		if err := tx.Model(&model.DeviceGroup{}).Where("id = ?", input.DeviceGroupId).Count(&count).Error; err != nil {
			return nil, err
		}
		if count == 0 {
			return nil, ErrDeploymentCodeInvalid
		}
	}
	platformJson, err := json.Marshal(platforms)
	if err != nil {
		return nil, err
	}
	plain, err := generateDeploymentCode()
	if err != nil {
		return nil, err
	}
	deploymentCode := &model.DeploymentCode{
		Name: input.Name, CodeHash: deploymentCodeHash(plain), CodeHint: deploymentCodeHint(plain),
		DeviceGroupId: input.DeviceGroupId, AllowedPlatforms: string(platformJson),
		ExpiresAt: input.ExpiresAt, MaxUses: input.MaxUses, Status: model.DeploymentCodeStatusActive,
		CreatedBy: actorUserId, RotatedFromId: input.RotatedFromId,
	}
	if err := tx.Create(deploymentCode).Error; err != nil {
		return nil, err
	}
	if err := tx.Create(&model.DeploymentAuditEvent{
		Action: action, DeploymentCodeId: deploymentCode.Id, ActorUserId: actorUserId, Ip: ip,
		Detail: "enrollment credential issued",
	}).Error; err != nil {
		return nil, err
	}
	return &DeploymentCodeSecret{DeploymentCode: deploymentCode, Code: plain}, nil
}

func (s *DeploymentCodeService) List(page, pageSize uint, status *int) (*model.DeploymentCodeList, error) {
	result := &model.DeploymentCodeList{}
	result.Page, result.PageSize = int64(page), int64(pageSize)
	tx := DB.Model(&model.DeploymentCode{})
	if status != nil {
		tx = tx.Where("status = ?", *status)
	}
	if err := tx.Count(&result.Total).Error; err != nil {
		return nil, err
	}
	if err := tx.Order("id desc").Scopes(Paginate(page, pageSize)).Find(&result.DeploymentCodes).Error; err != nil {
		return nil, err
	}
	return result, nil
}

func (s *DeploymentCodeService) Revoke(id, actorUserId uint, ip string) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		var code model.DeploymentCode
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&code, id).Error; err != nil {
			return err
		}
		if code.Status == model.DeploymentCodeStatusRevoked {
			return ErrDeploymentCodeRevoked
		}
		now := time.Now().Unix()
		if err := tx.Model(&code).Updates(map[string]interface{}{
			"status": model.DeploymentCodeStatusRevoked, "revoked_by": actorUserId, "revoked_at": now,
		}).Error; err != nil {
			return err
		}
		return tx.Create(&model.DeploymentAuditEvent{
			Action: model.DeploymentAuditRevoked, DeploymentCodeId: id, ActorUserId: actorUserId,
			Ip: ip, Detail: "enrollment credential revoked",
		}).Error
	})
}

func (s *DeploymentCodeService) Rotate(id, actorUserId uint, ip string) (*DeploymentCodeSecret, error) {
	var result *DeploymentCodeSecret
	err := DB.Transaction(func(tx *gorm.DB) error {
		var old model.DeploymentCode
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&old, id).Error; err != nil {
			return err
		}
		if old.Status != model.DeploymentCodeStatusActive || old.ExpiresAt <= time.Now().Unix() {
			return ErrDeploymentCodeInvalid
		}
		if err := tx.Model(&old).Updates(map[string]interface{}{
			"status": model.DeploymentCodeStatusRevoked, "revoked_by": actorUserId, "revoked_at": time.Now().Unix(),
		}).Error; err != nil {
			return err
		}
		created, err := s.createWithTx(tx, DeploymentCodeCreateInput{
			Name: old.Name, DeviceGroupId: old.DeviceGroupId, AllowedPlatforms: old.PlatformList(),
			ExpiresAt: old.ExpiresAt, MaxUses: old.MaxUses, RotatedFromId: old.Id,
		}, actorUserId, ip, model.DeploymentAuditRotated)
		if err != nil {
			return err
		}
		result = created
		return tx.Create(&model.DeploymentAuditEvent{
			Action: model.DeploymentAuditRotated, DeploymentCodeId: old.Id, ActorUserId: actorUserId,
			Ip: ip, Detail: fmt.Sprintf("rotated to deployment code %d", created.DeploymentCode.Id),
		}).Error
	})
	return result, err
}

func (s *DeploymentCodeService) Claim(input DeploymentClaimInput) (*model.Peer, error) {
	input.Code = normalizeDeploymentCode(input.Code)
	input.DeviceId = strings.TrimSpace(input.DeviceId)
	input.Uuid = strings.TrimSpace(input.Uuid)
	input.Platform = NormalizeDeploymentPlatform(input.Platform)
	if input.Code == "" || input.DeviceId == "" || input.Uuid == "" {
		return nil, ErrDeploymentCodeInvalid
	}
	if _, ok := validDeploymentPlatforms[input.Platform]; !ok {
		return nil, ErrDeploymentPlatform
	}
	hash := deploymentCodeHash(input.Code)
	if Lock != nil {
		Lock.Lock("deployment-code:" + hash)
		defer Lock.UnLock("deployment-code:" + hash)
	}

	var peer *model.Peer
	err := DB.Transaction(func(tx *gorm.DB) error {
		var code model.DeploymentCode
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("code_hash = ?", hash).First(&code).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrDeploymentCodeInvalid
			}
			return err
		}
		if code.Status != model.DeploymentCodeStatusActive {
			return ErrDeploymentCodeRevoked
		}
		if code.ExpiresAt <= time.Now().Unix() {
			return ErrDeploymentCodeExpired
		}
		if code.UsedCount >= code.MaxUses {
			return ErrDeploymentCodeExhausted
		}
		allowed := false
		for _, platform := range code.PlatformList() {
			if platform == input.Platform {
				allowed = true
				break
			}
		}
		if !allowed {
			return ErrDeploymentPlatform
		}

		var existing model.Peer
		findErr := tx.Where("id = ? OR uuid = ?", input.DeviceId, input.Uuid).First(&existing).Error
		if findErr == nil {
			if existing.Id == input.DeviceId && existing.Uuid == input.Uuid && existing.Status == model.PeerStatusPending && existing.GroupId == code.DeviceGroupId {
				peer = &existing
				return nil
			}
			return ErrDeviceAlreadyEnrolled
		}
		if !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return findErr
		}

		created := &model.Peer{
			Id: input.DeviceId, Uuid: input.Uuid, Hostname: strings.TrimSpace(input.Hostname),
			Os: strings.TrimSpace(input.Os), Username: strings.TrimSpace(input.Username),
			Version: strings.TrimSpace(input.Version), GroupId: code.DeviceGroupId,
		}
		if err := tx.Create(created).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.Peer{}).Where("row_id = ?", created.RowId).Update("status", model.PeerStatusPending).Error; err != nil {
			return err
		}
		created.Status = model.PeerStatusPending
		if err := tx.Model(&code).UpdateColumn("used_count", gorm.Expr("used_count + 1")).Error; err != nil {
			return err
		}
		if err := tx.Create(&model.DeploymentAuditEvent{
			Action: model.DeploymentAuditClaimed, DeploymentCodeId: code.Id, PeerRowId: created.RowId,
			DeviceId: created.Id, Uuid: created.Uuid, Ip: input.Ip, Detail: "device enrolled pending approval",
		}).Error; err != nil {
			return err
		}
		peer = created
		return nil
	})
	return peer, err
}

func (s *DeploymentCodeService) AuditList(codeId, page, pageSize uint) (*model.DeploymentAuditEventList, error) {
	result := &model.DeploymentAuditEventList{}
	result.Page, result.PageSize = int64(page), int64(pageSize)
	tx := DB.Model(&model.DeploymentAuditEvent{}).Where("deployment_code_id = ?", codeId)
	if err := tx.Count(&result.Total).Error; err != nil {
		return nil, err
	}
	if err := tx.Order("id desc").Scopes(Paginate(page, pageSize)).Find(&result.Events).Error; err != nil {
		return nil, err
	}
	return result, nil
}
