package model

import "encoding/json"

const (
	GroupTypeDefault = 1 // 默认
	GroupTypeShare   = 2 // 共享
)

type Group struct {
	IdModel
	Name string `json:"name" gorm:"default:'';not null;"`
	Type int    `json:"type" gorm:"default:1;not null;"`
	// RouteNames 组自定义导航路由名称列表，以 JSON 字符串数组存储（如 ["MyPeer","MyInfo"]）。
	// 空字符串表示未配置，使用系统默认导航（model.UserRouteNames）。
	// 仅用于控制前端导航可见性，不构成后端接口鉴权（详见 UserService.RouteNames 注释）。
	RouteNames string `json:"route_names" gorm:"default:'';not null;"`
	TimeModel
}

// RouteNameList 解析组配置的自定义导航路由名称列表。
// 未配置（空串）或内容非法时返回 nil，调用方应回退到默认导航。
func (g *Group) RouteNameList() []string {
	if g.RouteNames == "" {
		return nil
	}
	var names []string
	if err := json.Unmarshal([]byte(g.RouteNames), &names); err != nil || len(names) == 0 {
		return nil
	}
	return names
}

type GroupList struct {
	Groups []*Group `json:"list"`
	Pagination
}

type DeviceGroup struct {
	IdModel
	Name string `json:"name" gorm:"default:'';not null;"`
	TimeModel
}

type DeviceGroupList struct {
	DeviceGroups []*DeviceGroup `json:"list"`
	Pagination
}
