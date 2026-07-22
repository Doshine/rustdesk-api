package service

import (
	"github.com/lejianwen/rustdesk-api/v2/config"
	"github.com/lejianwen/rustdesk-api/v2/lib/cache"
	"github.com/lejianwen/rustdesk-api/v2/lib/jwt"
	"github.com/lejianwen/rustdesk-api/v2/lib/lock"
	"github.com/lejianwen/rustdesk-api/v2/model"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Service struct {
	//AdminService     *AdminService
	//AdminRoleService *AdminRoleService
	*UserService
	*AddressBookService
	*TagService
	*PeerService
	*GroupService
	*OauthService
	*LoginLogService
	*AuditService
	*ShareRecordService
	*ServerCmdService
	*RelayNodeService
	*LdapService
	*AppService
	*SmsService
	*MfaService
	*PasskeyService
}

type Dependencies struct {
	Config *config.Config
	DB     *gorm.DB
	Logger *log.Logger
	Jwt    *jwt.Jwt
	Lock   *lock.Locker
	Cache  cache.Handler
}

var Config *config.Config
var DB *gorm.DB
var Logger *log.Logger
var Jwt *jwt.Jwt
var Lock lock.Locker
var Cache cache.Handler

var AllService *Service

func New(c *config.Config, g *gorm.DB, l *log.Logger, j *jwt.Jwt, lo lock.Locker, ca cache.Handler) (*Service, error) {
	Config = c
	DB = g
	Logger = l
	Jwt = j
	Lock = lo
	Cache = ca
	AllService = new(Service)
	AllService.MfaService = &MfaService{}
	passkeyService, err := NewPasskeyServiceFromConfig(c)
	if err != nil {
		return nil, err
	}
	AllService.PasskeyService = passkeyService
	smsService, err := NewSmsServiceFromConfig(c, ca)
	if err != nil {
		return nil, err
	}
	AllService.SmsService = smsService
	return AllService, nil
}

func Paginate(page, pageSize uint) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if page == 0 {
			page = 1
		}
		if pageSize == 0 {
			pageSize = 10
		}
		offset := (page - 1) * pageSize
		return db.Offset(int(offset)).Limit(int(pageSize))
	}
}

func CommonEnable() func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("status = ?", model.COMMON_STATUS_ENABLE)
	}
}
