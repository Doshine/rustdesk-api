package service

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"sync"
	"time"

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
	mu     sync.Mutex
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

// NewSmsServiceFromConfig 根据配置创建短信服务，生产 provider 初始化失败时直接返回错误。
func NewSmsServiceFromConfig(c *config.Config, ca cache.Handler) (*SmsService, error) {
	cfg := &c.Sms
	sender, err := sms.NewSender(&sms.Config{
		Provider:        cfg.Provider,
		AccessKeyId:     cfg.AccessKeyId,
		AccessKeySecret: cfg.AccessKeySecret,
		SignName:        cfg.SignName,
		TemplateCode:    cfg.TemplateCode,
		Endpoint:        cfg.Endpoint,
	}, Logger)
	if err != nil {
		return nil, err
	}
	if ca == nil {
		ca, err = newSmsCache(c)
		if err != nil {
			return nil, err
		}
	}
	return NewSmsService(sender, ca), nil
}

// newSmsCache 短信验证码缓存, 与 global.Cache 同样的配置来源, 缺省为内存缓存
func newSmsCache(c *config.Config) (cache.Handler, error) {
	switch c.Cache.Type {
	case cache.TypeFile:
		fc := cache.NewFileCache()
		fc.SetDir(c.Cache.FileDir)
		return fc, nil
	case cache.TypeRedis:
		opts, err := c.Cache.RedisOptions()
		if err != nil {
			return nil, err
		}
		return cache.NewRedisWithPrefix(opts, c.Cache.RedisKeyPrefix), nil
	default:
		return cache.NewMemoryCache(0), nil
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
func generateSmsCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
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

func (ss *SmsService) increment(key string, expiration int) (int64, error) {
	if atomicCache, ok := ss.cache.(cache.AtomicHandler); ok {
		return atomicCache.Increment(key, expiration)
	}
	ss.mu.Lock()
	defer ss.mu.Unlock()
	count := ss.getCount(key) + 1
	if err := ss.cache.Set(key, count, expiration); err != nil {
		return 0, err
	}
	return int64(count), nil
}

func (ss *SmsService) decrement(key string, expiration int) {
	if atomicCache, ok := ss.cache.(cache.AtomicHandler); ok {
		_ = atomicCache.Decrement(key)
		return
	}
	ss.mu.Lock()
	defer ss.mu.Unlock()
	count := ss.getCount(key)
	if count > 0 {
		_ = ss.cache.Set(key, count-1, expiration)
	}
}

// SendLoginCode 发送登录验证码
func (ss *SmsService) SendLoginCode(phone, ip string) error {
	if !IsValidCnPhone(phone) {
		return errors.New("PhoneFormatError")
	}
	//单手机号日限
	dailyKey := ss.dailyLimitKey(phone)
	now := time.Now()
	endOfDay := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
	dailyExpiration := int(endOfDay.Sub(now).Seconds()) + 1
	ipKey := ss.ipLimitKey(ip)
	dailyReserved := false
	ipReserved := false
	rollbackLimits := func() {
		if dailyReserved {
			ss.decrement(dailyKey, dailyExpiration)
		}
		if ipReserved {
			ss.decrement(ipKey, 60)
		}
	}
	if Config.Sms.DailyLimit > 0 {
		count, err := ss.increment(dailyKey, dailyExpiration)
		if err != nil {
			return errors.New("SmsSendFailed")
		}
		dailyReserved = true
		if count > int64(Config.Sms.DailyLimit) {
			rollbackLimits()
			return errors.New("SmsDailyLimitExceeded")
		}
	}
	//单IP分钟限
	if ip != "" && Config.Sms.PerIpLimit > 0 {
		count, err := ss.increment(ipKey, 60)
		if err != nil {
			rollbackLimits()
			return errors.New("SmsSendFailed")
		}
		ipReserved = true
		if count > int64(Config.Sms.PerIpLimit) {
			rollbackLimits()
			return errors.New("SmsIpLimitExceeded")
		}
	}

	code, err := generateSmsCode()
	if err != nil {
		Logger.Error("secure SMS code generation failed: ", err)
		rollbackLimits()
		return errors.New("SmsSendFailed")
	}
	if err := ss.cache.Set(smsCodeKeyPrefix+phone, code, ss.codeExpireSeconds()); err != nil {
		Logger.Warn("sms cache set code error: ", err)
		rollbackLimits()
		return errors.New("SmsSendFailed")
	}

	if err := ss.sender.SendCode(phone, code); err != nil {
		Logger.Warn("sms send code error: ", err)
		//发送失败则作废已存验证码, 不消耗限流配额
		_ = ss.cache.Set(smsCodeKeyPrefix+phone, "", 1)
		rollbackLimits()
		return errors.New("SmsSendFailed")
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
	var err error
	if atomicCache, ok := ss.cache.(cache.AtomicHandler); ok {
		err = atomicCache.GetAndDelete(key, &saved)
	} else {
		ss.mu.Lock()
		defer ss.mu.Unlock()
		err = ss.cache.Get(key, &saved)
		if err == nil {
			_ = ss.cache.Set(key, "", 1)
		}
	}
	if err != nil {
		Logger.Warn("sms cache get code error: ", err)
		return false
	}
	if saved == "" {
		return false
	}
	return saved == code
}
