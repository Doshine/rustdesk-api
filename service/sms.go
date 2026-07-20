package service

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/lejianwen/rustdesk-api/v2/config"
	"github.com/lejianwen/rustdesk-api/v2/lib/cache"
	"github.com/lejianwen/rustdesk-api/v2/lib/sms"
)

// SmsCodeExpireDefault 短信验证码默认有效期(秒)
const SmsCodeExpireDefault = 300

const (
	smsCodeKeyPrefix  = "sms:code:"
	smsDailyKeyPrefix = "sms:day:"
	smsIpKeyPrefix    = "sms:ip:"
)

// 中国大陆手机号 1[3-9]\d{9}
var cnPhoneRegexp = regexp.MustCompile(`^1[3-9]\d{9}$`)

type SmsService struct {
	sender sms.Sender
	cache  cache.Handler
}

// NewSmsService 创建短信服务
func NewSmsService(sender sms.Sender, c cache.Handler) *SmsService {
	if c == nil {
		c = cache.NewMemoryCache(0)
	}
	return &SmsService{
		sender: sender,
		cache:  c,
	}
}

// NewSmsServiceFromConfig 根据配置创建短信服务, sender 初始化失败时降级为mock
func NewSmsServiceFromConfig(c *config.Config) *SmsService {
	cfg := &c.Sms
	sender := sms.NewSender(&sms.Config{
		Provider:        cfg.Provider,
		AccessKeyId:     cfg.AccessKeyId,
		AccessKeySecret: cfg.AccessKeySecret,
		SignName:        cfg.SignName,
		TemplateCode:    cfg.TemplateCode,
		Endpoint:        cfg.Endpoint,
	}, Logger)
	return NewSmsService(sender, newSmsCache(c))
}

// newSmsCache 短信验证码缓存, 与 global.Cache 同样的配置来源, 缺省为内存缓存
func newSmsCache(c *config.Config) cache.Handler {
	switch c.Cache.Type {
	case cache.TypeFile:
		fc := cache.NewFileCache()
		fc.SetDir(c.Cache.FileDir)
		return fc
	case cache.TypeRedis:
		return cache.NewRedis(&redis.Options{
			Addr:     c.Cache.RedisAddr,
			Password: c.Cache.RedisPwd,
			DB:       c.Cache.RedisDb,
		})
	default:
		return cache.NewMemoryCache(0)
	}
}

// IsValidCnPhone 校验中国大陆手机号格式
func IsValidCnPhone(phone string) bool {
	return cnPhoneRegexp.MatchString(phone)
}

// MaskCnPhone 手机号掩码, 如 138****5678
func MaskCnPhone(phone string) string {
	if len(phone) != 11 {
		return phone
	}
	return phone[:3] + "****" + phone[7:]
}

// generateSmsCode 生成6位数字验证码
func generateSmsCode() string {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
	}
	return fmt.Sprintf("%06d", n.Int64())
}

// codeExpireSeconds 验证码有效期(秒)
func (ss *SmsService) codeExpireSeconds() int {
	if Config.Sms.CodeExpireSeconds > 0 {
		return Config.Sms.CodeExpireSeconds
	}
	return SmsCodeExpireDefault
}

// dailyLimitKey 单手机号日限计数key
func (ss *SmsService) dailyLimitKey(phone string) string {
	return smsDailyKeyPrefix + phone + ":" + time.Now().Format("20060102")
}

// ipLimitKey 单IP分钟限计数key
func (ss *SmsService) ipLimitKey(ip string) string {
	return smsIpKeyPrefix + ip + ":" + time.Now().Format("200601021504")
}

func (ss *SmsService) getCount(key string) int {
	count := 0
	if err := ss.cache.Get(key, &count); err != nil {
		Logger.Warn("sms cache get count error: ", err)
	}
	return count
}

// SendLoginCode 发送登录验证码
func (ss *SmsService) SendLoginCode(phone, ip string) error {
	if !IsValidCnPhone(phone) {
		return errors.New("PhoneFormatError")
	}
	//单手机号日限
	dailyKey := ss.dailyLimitKey(phone)
	if Config.Sms.DailyLimit > 0 && ss.getCount(dailyKey) >= Config.Sms.DailyLimit {
		return errors.New("SmsDailyLimitExceeded")
	}
	//单IP分钟限
	ipKey := ss.ipLimitKey(ip)
	if ip != "" && Config.Sms.PerIpLimit > 0 && ss.getCount(ipKey) >= Config.Sms.PerIpLimit {
		return errors.New("SmsIpLimitExceeded")
	}

	code := generateSmsCode()
	if err := ss.cache.Set(smsCodeKeyPrefix+phone, code, ss.codeExpireSeconds()); err != nil {
		Logger.Warn("sms cache set code error: ", err)
		return errors.New("SmsSendFailed")
	}

	if err := ss.sender.SendCode(phone, code); err != nil {
		Logger.Warn("sms send code error: ", err)
		//发送失败则作废已存验证码, 不消耗限流配额
		_ = ss.cache.Set(smsCodeKeyPrefix+phone, "", 1)
		return errors.New("SmsSendFailed")
	}

	//发送成功后计数, 日限计数当日有效, IP限计数60秒有效
	now := time.Now()
	endOfDay := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
	_ = ss.cache.Set(dailyKey, ss.getCount(dailyKey)+1, int(endOfDay.Sub(now).Seconds())+1)
	if ip != "" {
		_ = ss.cache.Set(ipKey, ss.getCount(ipKey)+1, 60)
	}
	return nil
}

// VerifyLoginCode 校验登录验证码, 一次即焚(无论对错都删除)
func (ss *SmsService) VerifyLoginCode(phone, code string) bool {
	if phone == "" || code == "" {
		return false
	}
	key := smsCodeKeyPrefix + phone
	saved := ""
	if err := ss.cache.Get(key, &saved); err != nil {
		Logger.Warn("sms cache get code error: ", err)
		return false
	}
	if saved == "" {
		return false
	}
	//无论对错都作废
	_ = ss.cache.Set(key, "", 1)
	return saved == code
}
