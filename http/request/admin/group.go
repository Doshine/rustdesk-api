package admin

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/lejianwen/rustdesk-api/v2/model"
)

type GroupForm struct {
	Id   uint   `json:"id"`
	Name string `json:"name" validate:"required"`
	Type int    `json:"type"`
	// RouteNames 组自定义导航（P3-1 角色导航）：JSON 字符串数组，
	// 如 "[\"MyPeer\",\"MyInfo\"]"；空字符串表示使用系统默认导航。
	// 仅控制前端导航可见性，不影响后端接口鉴权。
	RouteNames string `json:"route_names"`
}

// ValidateRouteNames 校验 RouteNames：允许空串（默认导航），否则必须是 JSON 字符串数组
func (gf *GroupForm) ValidateRouteNames() error {
	s := strings.TrimSpace(gf.RouteNames)
	if s == "" {
		return nil
	}
	var names []string
	if err := json.Unmarshal([]byte(s), &names); err != nil {
		return errors.New("route_names must be a JSON array of strings")
	}
	return nil
}

// normalizeRouteNames 归一化 RouteNames：去空白并压缩 JSON；非法输入归一为空串（默认导航）
func normalizeRouteNames(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	var names []string
	if err := json.Unmarshal([]byte(s), &names); err != nil {
		return ""
	}
	b, err := json.Marshal(names)
	if err != nil {
		return ""
	}
	return string(b)
}

func (gf *GroupForm) FromGroup(group *model.Group) *GroupForm {
	gf.Id = group.Id
	gf.Name = group.Name
	gf.Type = group.Type
	gf.RouteNames = group.RouteNames
	return gf
}

func (gf *GroupForm) ToGroup() *model.Group {
	group := &model.Group{}
	group.Id = gf.Id
	group.Name = gf.Name
	group.Type = gf.Type
	group.RouteNames = normalizeRouteNames(gf.RouteNames)
	return group
}

type DeviceGroupForm struct {
	Id   uint   `json:"id"`
	Name string `json:"name" validate:"required"`
}

func (gf *DeviceGroupForm) ToDeviceGroup() *model.DeviceGroup {
	group := &model.DeviceGroup{}
	group.Id = gf.Id
	group.Name = gf.Name
	return group
}
