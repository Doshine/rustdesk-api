package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/config"
	"github.com/lejianwen/rustdesk-api/v2/global"
	"github.com/lejianwen/rustdesk-api/v2/http"
	"github.com/lejianwen/rustdesk-api/v2/lib/cache"
	"github.com/lejianwen/rustdesk-api/v2/lib/jwt"
	"github.com/lejianwen/rustdesk-api/v2/lib/lock"
	"github.com/lejianwen/rustdesk-api/v2/lib/logger"
	"github.com/lejianwen/rustdesk-api/v2/lib/orm"
	"github.com/lejianwen/rustdesk-api/v2/lib/upload"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
	"github.com/lejianwen/rustdesk-api/v2/utils"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

const DatabaseVersion = 273

// @title 管理系统API
// @version 1.0
// @description 接口
// @basePath /api
// @securityDefinitions.apikey token
// @in header
// @name api-token
// @securitydefinitions.apikey BearerAuth
// @in header
// @name Authorization

var rootCmd = &cobra.Command{
	Use:   "apimain",
	Short: "RUSTDESK API SERVER",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		InitGlobal()
	},
	Run: func(cmd *cobra.Command, args []string) {
		global.Logger.Info("API SERVER START")
		http.ApiInit()
	},
}

var resetPwdCmd = &cobra.Command{
	Use:     "reset-admin-pwd [pwd]",
	Example: "reset-admin-pwd 123456",
	Short:   "Reset Admin Password",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		pwd := args[0]
		admin := service.AllService.UserService.InfoById(1)
		if admin.Id == 0 {
			global.Logger.Warn("user not found! ")
			return
		}
		err := service.AllService.UserService.UpdatePassword(admin, pwd)
		if err != nil {
			global.Logger.Error("reset password fail! ", err)
			return
		}
		global.Logger.Info("reset password success! ")
	},
}
var resetUserPwdCmd = &cobra.Command{
	Use:     "reset-pwd [userId] [pwd]",
	Example: "reset-pwd 2 123456",
	Short:   "Reset User Password",
	Args:    cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		userId := args[0]
		pwd := args[1]
		uid, err := strconv.Atoi(userId)
		if err != nil {
			global.Logger.Warn("userId must be int!")
			return
		}
		if uid <= 0 {
			global.Logger.Warn("userId must be greater than 0! ")
			return
		}
		u := service.AllService.UserService.InfoById(uint(uid))
		if u.Id == 0 {
			global.Logger.Warn("user not found! ")
			return
		}
		err = service.AllService.UserService.UpdatePassword(u, pwd)
		if err != nil {
			global.Logger.Warn("reset password fail! ", err)
			return
		}
		global.Logger.Info("reset password success!")
	},
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&global.ConfigPath, "config", "c", "./conf/config.yaml", "choose config file")
	rootCmd.AddCommand(resetPwdCmd, resetUserPwdCmd)
}
func main() {
	if err := rootCmd.Execute(); err != nil {
		global.Logger.Error(err)
		os.Exit(1)
	}
}

func InitGlobal() {
	//配置解析
	global.Viper = config.Init(&global.Config, global.ConfigPath)

	//日志
	global.Logger = logger.New(&logger.Config{
		Path:         global.Config.Logger.Path,
		Level:        global.Config.Logger.Level,
		ReportCaller: global.Config.Logger.ReportCaller,
	})

	global.InitI18n()

	//cache
	if global.Config.Cache.Type == cache.TypeFile {
		fc := cache.NewFileCache()
		fc.SetDir(global.Config.Cache.FileDir)
		global.Cache = fc
	} else if global.Config.Cache.Type == cache.TypeRedis {
		redisOptions, err := global.Config.Cache.RedisOptions()
		if err != nil {
			global.Logger.Fatalf("invalid Redis configuration: %v", err)
		}
		redisCache := cache.NewRedisWithPrefix(redisOptions, global.Config.Cache.RedisKeyPrefix)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := redisCache.Ping(ctx); err != nil {
			_ = redisCache.Close()
			global.Logger.Fatalf("Redis readiness check failed: %v", err)
		}
		global.Cache = redisCache
	} else {
		global.Cache = cache.NewMemoryCache(0)
	}
	//gorm
	if global.Config.Gorm.Type == config.TypeMysql {

		dsn := fmt.Sprintf("%s:%s@(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local&tls=%s",
			global.Config.Mysql.Username,
			global.Config.Mysql.Password,
			global.Config.Mysql.Addr,
			global.Config.Mysql.Dbname,
			global.Config.Mysql.Tls,
		)

		global.DB = orm.NewMysql(&orm.MysqlConfig{
			Dsn:          dsn,
			MaxIdleConns: global.Config.Gorm.MaxIdleConns,
			MaxOpenConns: global.Config.Gorm.MaxOpenConns,
		}, global.Logger)
	} else if global.Config.Gorm.Type == config.TypePostgresql {
		dsn, err := global.Config.Postgresql.DSN()
		if err != nil {
			global.Logger.Fatalf("invalid PostgreSQL connection configuration: %v", err)
		}
		global.DB, err = orm.NewPostgresql(&orm.PostgresqlConfig{
			Dsn:          dsn,
			TimeZone:     global.Config.Postgresql.TimeZone,
			MaxIdleConns: global.Config.Gorm.MaxIdleConns,
			MaxOpenConns: global.Config.Gorm.MaxOpenConns,
		}, global.Logger)
		if err != nil {
			global.Logger.Fatalf("PostgreSQL initialization failed: %v", err)
		}
	} else {
		//sqlite
		global.DB = orm.NewSqlite(&orm.SqliteConfig{
			MaxIdleConns: global.Config.Gorm.MaxIdleConns,
			MaxOpenConns: global.Config.Gorm.MaxOpenConns,
		}, global.Logger)
	}
	DatabaseAutoUpdate()
	BootstrapInitialAdmin()

	//validator
	global.ApiInitValidator()

	//oss
	global.Oss = &upload.Oss{
		AccessKeyId:     global.Config.Oss.AccessKeyId,
		AccessKeySecret: global.Config.Oss.AccessKeySecret,
		Host:            global.Config.Oss.Host,
		CallbackUrl:     global.Config.Oss.CallbackUrl,
		ExpireTime:      global.Config.Oss.ExpireTime,
		MaxByte:         global.Config.Oss.MaxByte,
	}

	//jwt
	//fmt.Println(global.Config.Jwt.PrivateKey)
	global.Jwt = jwt.NewJwt(global.Config.Jwt.Key, global.Config.Jwt.ExpireDuration)
	//locker
	global.Lock = lock.NewLocal()

	//service
	if _, err := service.New(&global.Config, global.DB, global.Logger, global.Jwt, global.Lock, global.Cache); err != nil {
		global.Logger.Fatalf("service initialization failed: %v", err)
	}

	global.LoginLimiter = utils.NewLoginLimiter(utils.SecurityPolicy{
		CaptchaThreshold: global.Config.App.CaptchaThreshold,
		BanThreshold:     global.Config.App.BanThreshold,
		AttemptsWindow:   10 * time.Minute,
		BanDuration:      30 * time.Minute,
	})
	global.LoginLimiter.RegisterProvider(utils.B64StringCaptchaProvider{})
}

// BootstrapInitialAdmin 在数据库尚无任何用户时创建初始 owner 账号与默认分组。
//
// 必须独立于 AutoMigrate 执行：显式迁移路径（gorm.auto-migrate=false，生产部署的
// 标准形态）下 DatabaseAutoUpdate 会提前 return，根本走不到里面的引导逻辑；
// 而且迁移脚本自身已向 versions 写入记录，那里的 versionCount==0 判据在该路径下
// 永远不成立。两者叠加的后果是：全新生产部署没有任何可登录账号。
//
// 这里改用"用户表为空"作为判据——那才是真正关心的不变量。
// 已有用户时直接返回，因此对既有的 AutoMigrate 路径是无副作用的空操作。
func BootstrapInitialAdmin() {
	db := global.DB
	if db == nil {
		return
	}
	var userCount int64
	if err := db.Model(&model.User{}).Count(&userCount).Error; err != nil {
		global.Logger.Fatalf("bootstrap initial admin: count users failed: %v", err)
	}
	if userCount > 0 {
		return
	}

	pwd := os.Getenv("RUSTDESK_API_BOOTSTRAP_ADMIN_PASSWORD")
	if len(pwd) < 16 {
		global.Logger.Fatal("database has no user yet; RUSTDESK_API_BOOTSTRAP_ADMIN_PASSWORD must contain at least 16 characters for first startup")
	}
	hashedPassword, err := utils.EncryptPassword(pwd)
	if err != nil {
		global.Logger.Fatalf("bootstrap initial admin: hash password failed: %v", err)
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		localizer := global.Localizer("")
		defaultName, _ := localizer.LocalizeMessage(&i18n.Message{ID: "DefaultGroup"})
		shareName, _ := localizer.LocalizeMessage(&i18n.Message{ID: "ShareGroup"})

		// 分组可能已由其他路径建过，按类型取用而不是无条件新建
		group := &model.Group{}
		if err := tx.Where("type = ?", model.GroupTypeDefault).First(group).Error; err != nil {
			group = &model.Group{Name: defaultName, Type: model.GroupTypeDefault}
			if err := tx.Create(group).Error; err != nil {
				return err
			}
		}
		shareGroup := &model.Group{}
		if err := tx.Where("type = ?", model.GroupTypeShare).First(shareGroup).Error; err != nil {
			shareGroup = &model.Group{Name: shareName, Type: model.GroupTypeShare}
			if err := tx.Create(shareGroup).Error; err != nil {
				return err
			}
		}

		isAdmin := true
		admin := &model.User{
			Username: "admin",
			Nickname: "Admin",
			Status:   model.COMMON_STATUS_ENABLE,
			IsAdmin:  &isAdmin,
			GroupId:  group.Id,
			Role:     model.RoleOwner,
			Password: hashedPassword,
		}
		return tx.Create(admin).Error
	})
	if err != nil {
		global.Logger.Fatalf("bootstrap initial admin: %v", err)
	}
	global.Logger.Info("initial admin created with externally supplied bootstrap password")
}

func DatabaseAutoUpdate() {
	version := DatabaseVersion

	db := global.DB
	if !global.Config.Gorm.AutoMigrate {
		if err := VerifyDatabaseVersion(db, uint(version)); err != nil {
			global.Logger.Fatalf("database schema validation failed: %v", err)
		}
		global.Logger.Infof("database schema version %d verified; startup AutoMigrate is disabled", version)
		return
	}

	if global.Config.Gorm.Type == config.TypeMysql {
		//检查存不存在数据库，不存在则创建
		dbName := db.Migrator().CurrentDatabase()
		if dbName == "" {
			dbName = global.Config.Mysql.Dbname
			// 移除 DSN 中的数据库名称，以便初始连接时不指定数据库
			dsnWithoutDB := fmt.Sprintf("%s:%s@(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
				global.Config.Mysql.Username,
				global.Config.Mysql.Password,
				global.Config.Mysql.Addr,
				"",
			)

			//新链接
			dbWithoutDB := orm.NewMysql(&orm.MysqlConfig{
				Dsn: dsnWithoutDB,
			}, global.Logger)
			// 获取底层的 *sql.DB 对象，并确保在程序退出时关闭连接
			sqlDBWithoutDB, err := dbWithoutDB.DB()
			if err != nil {
				global.Logger.Fatalf("获取底层 *sql.DB 对象失败: %v", err)
			}
			defer func() {
				if err := sqlDBWithoutDB.Close(); err != nil {
					global.Logger.Errorf("关闭连接失败: %v", err)
				}
			}()

			err = dbWithoutDB.Exec("CREATE DATABASE IF NOT EXISTS " + dbName + " DEFAULT CHARSET utf8mb4").Error
			if err != nil {
				global.Logger.Fatalf("create database failed: %v", err)
			}
		}
	}

	if !db.Migrator().HasTable(&model.Version{}) {
		if err := Migrate(uint(version)); err != nil {
			global.Logger.Fatalf("database migration failed: %v", err)
		}
	} else {
		//查找最后一个version
		var v model.Version
		db.Last(&v)
		if v.Version < uint(version) {
			if err := Migrate(uint(version)); err != nil {
				global.Logger.Fatalf("database migration failed: %v", err)
			}
		}

		// 245迁移
		if v.Version < 245 {
			//oauths 表的 oauth_type 字段设置为 op同样的值
			db.Exec("update oauths set oauth_type = op")
			db.Exec("update oauths set issuer = 'https://accounts.google.com' where op = 'google'")
			db.Exec("update user_thirds set oauth_type = third_type, op = third_type")
			//通过email迁移旧的google授权
			uts := make([]model.UserThird, 0)
			db.Where("oauth_type = ?", "google").Find(&uts)
			for _, ut := range uts {
				if ut.UserId > 0 {
					db.Model(&model.User{}).Where("id = ?", ut.UserId).Update("email", ut.OpenId)
				}
			}
		}
		if v.Version < 246 {
			db.Exec("update oauths set issuer = 'https://accounts.google.com' where op = 'google' and issuer is null")
		}
		if v.Version < 267 {
			// P3-1 设备审批：存量设备审批状态统一置为"已通过"，保持升级前行为一致。
			// peers.status 列由 AutoMigrate 以 default:1 添加，此处为兜底，
			// 确保任何存量的 0 / NULL 值都被纠正为已通过（仅在升级到 267 时执行一次）。
			db.Exec("update peers set status = ? where status <> ? or status is null", model.PeerStatusApproved, model.PeerStatusApproved)
		}
	}

}

func VerifyDatabaseVersion(db *gorm.DB, expected uint) error {
	if !db.Migrator().HasTable(&model.Version{}) {
		return fmt.Errorf("versions table is missing; run yinhe/deploy/migrate.sh to apply migrations for version %d before startup", expected)
	}

	// 取 MAX(version) 而不是「按 id 倒序取第一条」：后者依赖 id 与 version
	// 严格同序，一旦有补丁迁移插入历史版本号就会读到错误的值。
	var current uint
	if err := db.Model(&model.Version{}).Select("coalesce(max(version), 0)").Scan(&current).Error; err != nil {
		return fmt.Errorf("read database version: %w", err)
	}
	if current == expected {
		return nil
	}
	// 区分两个方向，给出可直接执行的下一步；原来只报「不匹配」，
	// 运维看到后并不知道该升级数据库还是回滚应用。
	if current < expected {
		return fmt.Errorf(
			"database version %d is older than application version %d; "+
				"run yinhe/deploy/migrate.sh (with a DDL-capable role) to apply pending migrations",
			current, expected)
	}
	return fmt.Errorf(
		"database version %d is newer than application version %d; "+
			"the database was migrated by a newer build — deploy the matching application version, "+
			"or roll the schema back with the corresponding down script in rustdesk-api/supabase/rollback/",
		current, expected)
}

func Migrate(version uint) error {
	global.Logger.Info("Migrating....", version)
	return global.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.AutoMigrate(
			&model.Version{},
			&model.User{},
			&model.UserToken{},
			&model.Tag{},
			&model.AddressBook{},
			&model.Peer{},
			&model.Group{},
			&model.UserThird{},
			&model.PasskeyCredential{},
			&model.Oauth{},
			&model.LoginLog{},
			&model.ShareRecord{},
			&model.AuditConn{},
			&model.AuditFile{},
			&model.AddressBookCollection{},
			&model.AddressBookCollectionRule{},
			&model.ServerCmd{},
			&model.DeviceGroup{},
			&model.RelayNode{},
			&model.DeploymentCode{},
			&model.DeploymentAuditEvent{},
		); err != nil {
			return err
		}
		// 269: backfill the compatibility role for existing accounts. Existing
		// administrators remain administrators; all other accounts become users.
		var users []model.User
		if err := tx.Where("role = '' OR role IS NULL").Find(&users).Error; err != nil {
			return err
		}
		for i := range users {
			role := model.RoleUser
			if users[i].IsAdmin != nil && *users[i].IsAdmin {
				role = model.RoleAdmin
			}
			if err := tx.Model(&model.User{}).Where("id = ?", users[i].Id).Update("role", role).Error; err != nil {
				return err
			}
		}

		var versionCount int64
		if err := tx.Model(&model.Version{}).Count(&versionCount).Error; err != nil {
			return err
		}
		if versionCount == 0 {
			pwd := os.Getenv("RUSTDESK_API_BOOTSTRAP_ADMIN_PASSWORD")
			if len(pwd) < 16 {
				return fmt.Errorf("RUSTDESK_API_BOOTSTRAP_ADMIN_PASSWORD must contain at least 16 characters for first startup")
			}
			hashedPassword, err := utils.EncryptPassword(pwd)
			if err != nil {
				return fmt.Errorf("hash bootstrap admin password: %w", err)
			}

			localizer := global.Localizer("")
			defaultGroup, _ := localizer.LocalizeMessage(&i18n.Message{
				ID: "DefaultGroup",
			})
			group := &model.Group{
				Name: defaultGroup,
				Type: model.GroupTypeDefault,
			}
			if err := tx.Create(group).Error; err != nil {
				return err
			}

			shareGroup, _ := localizer.LocalizeMessage(&i18n.Message{
				ID: "ShareGroup",
			})
			groupShare := &model.Group{
				Name: shareGroup,
				Type: model.GroupTypeShare,
			}
			if err := tx.Create(groupShare).Error; err != nil {
				return err
			}
			//是true
			is_admin := true
			admin := &model.User{
				Username: "admin",
				Nickname: "Admin",
				Status:   model.COMMON_STATUS_ENABLE,
				IsAdmin:  &is_admin,
				GroupId:  group.Id,
				Role:     model.RoleOwner,
			}
			admin.Password = hashedPassword
			if err := tx.Create(admin).Error; err != nil {
				return err
			}
			global.Logger.Info("initial admin created with externally supplied bootstrap password")
		}
		return tx.Create(&model.Version{Version: version}).Error
	})
}
