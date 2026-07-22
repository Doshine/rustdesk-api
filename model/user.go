package model

type User struct {
	IdModel
	Username string `json:"username" gorm:"default:'';not null;uniqueIndex"`
	Email    string `json:"email" gorm:"default:'';not null;index"`
	Phone    string `json:"phone" gorm:"default:'';not null;index"`
	// Email	string     	`json:"email" `
	Password                string `json:"-" gorm:"default:'';not null;"`
	Nickname                string `json:"nickname" gorm:"default:'';not null;"`
	Avatar                  string `json:"avatar" gorm:"default:'';not null;"`
	GroupId                 uint   `json:"group_id" gorm:"default:0;not null;index"`
	IsAdmin                 *bool  `json:"is_admin" gorm:"default:0;not null;"`
	Role                    string `json:"role" gorm:"default:'user';not null;index"`
	MfaEnabled              bool   `json:"mfa_enabled" gorm:"default:0;not null;index"`
	MfaSecretEncrypted      string `json:"-" gorm:"default:'';not null;"`
	MfaBackupCodesEncrypted string `json:"-" gorm:"default:'';not null;"`
	// WebAuthnUserHandle is an opaque, stable per-account handle. It is not a
	// credential secret and is kept separately from each passkey row so all
	// discoverable credentials for one account resolve to the same user.
	WebAuthnUserHandle string     `json:"-" gorm:"type:varchar(128);default:'';not null;index"`
	Status             StatusCode `json:"status" gorm:"default:1;not null;"`
	Remark             string     `json:"remark" gorm:"default:'';not null;"`
	TimeModel
}

// BeforeSave 钩子用于确保 email 字段有合理的默认值
//func (u *User) BeforeSave(tx *gorm.DB) (err error) {
//	// 如果 email 为空，设置为默认值
//	if u.Email == "" {
//		u.Email = fmt.Sprintf("%s@example.com", u.Username)
//	}
//	return nil
//}

type UserList struct {
	Users []*User `json:"list,omitempty"`
	Pagination
}

var UserRouteNames = []string{
	"MyTagList", "MyAddressBookList", "MyInfo", "MyAddressBookCollection", "MyPeer", "MyShareRecordList", "MyLoginLog",
}
var AdminRouteNames = []string{"*"}
