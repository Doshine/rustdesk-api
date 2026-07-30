package api

import (
	"strings"
	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/global"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/http/response/api"
	"github.com/lejianwen/rustdesk-api/v2/service"
	"time"
)

type WebClient struct {
}

// ServerConfig 服务配置
// @Tags WEBCLIENT
// @Summary 服务配置
// @Description 服务配置,给webclient提供api-server
// @Accept  json
// @Produce  json
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /server-config [get]
// @Security token
func (i *WebClient) ServerConfig(c *gin.Context) {
	u := service.AllService.UserService.CurUser(c)

	peers := map[string]*api.WebClientPeerPayload{}
	abs := service.AllService.AddressBookService.ListByUserIdAndCollectionId(u.Id, 0, 1, 100)
	for _, ab := range abs.AddressBooks {
		pp := &api.WebClientPeerPayload{}
		pp.FromAddressBook(ab)
		peers[ab.Id] = pp
	}
	response.Success(
		c,
		gin.H{
			"id_server": global.Config.Rustdesk.IdServer,
			"key":       global.Config.Rustdesk.Key,
			"peers":     peers,
		},
	)
}

// SharedPeer 分享的peer
// @Tags WEBCLIENT
// @Summary 分享的peer
// @Description 分享的peer
// @Accept  json
// @Produce  json
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /shared-peer [post]
// sharedPeerMaxLifetime 是分享链接的绝对有效期上限。
//
// 原实现里 Expire==0 表示"永不过期"，而本接口是**匿名**的：拿到 share_token
// 即可换取目标设备的远控密码与服务端 key。链接一旦外流（聊天记录、邮件、
// 截图）就等同于永久有效的凭据。这里给出硬上限，即使 Expire==0 也会到期。
const sharedPeerMaxLifetime = 24 * time.Hour

func (i *WebClient) SharedPeer(c *gin.Context) {
	j := &gin.H{}
	if err := c.ShouldBindJSON(j); err != nil {
		response.Fail(c, 101, "share_token is required")
		return
	}
	// 原实现是 (*j)["share_token"].(string) —— 未检查的类型断言。
	// share_token 缺失或不是字符串时直接 panic；虽然被 gin.Recovery() 兜住，
	// 但每个畸形请求都会产生一次 panic 与完整栈日志，是廉价的 DoS 与日志淹没面。
	raw, exists := (*j)["share_token"]
	if !exists {
		response.Fail(c, 101, "share_token is required")
		return
	}
	t, okStr := raw.(string)
	if !okStr || strings.TrimSpace(t) == "" {
		response.Fail(c, 101, "share_token is required")
		return
	}
	sr := service.AllService.AddressBookService.SharedPeer(t)
	if sr == nil || sr.Id == 0 {
		response.Fail(c, 101, "share not found")
		return
	}
	// 过期判定：取「显式 Expire」与「绝对上限」中较早的一个。
	ca := time.Time(sr.CreatedAt)
	deadline := ca.Add(sharedPeerMaxLifetime)
	if sr.Expire != 0 {
		if explicit := ca.Add(time.Second * time.Duration(sr.Expire)); explicit.Before(deadline) {
			deadline = explicit
		}
	}
	if deadline.Before(time.Now()) {
		response.Fail(c, 101, "share expired")
		return
	}

	ab := service.AllService.AddressBookService.InfoByUserIdAndId(sr.UserId, sr.PeerId)
	if ab.RowId == 0 {
		response.Fail(c, 101, "peer not found")
		return
	}
	pp := &api.WebClientPeerPayload{}
	pp.FromShareRecord(sr)
	pp.Info.Username = ab.Username
	pp.Info.Hostname = ab.Hostname
	response.Success(c, gin.H{
		"id_server": global.Config.Rustdesk.IdServer,
		"key":       global.Config.Rustdesk.Key,
		"peer":      pp,
	})
}

// ServerConfigV2 服务配置
// @Tags WEBCLIENT_V2
// @Summary 服务配置
// @Description 服务配置,给webclient提供api-server
// @Accept  json
// @Produce  json
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /server-config-v2 [get]
// @Security token
func (i *WebClient) ServerConfigV2(c *gin.Context) {
	response.Success(
		c,
		gin.H{
			"id_server": global.Config.Rustdesk.IdServer,
			"key":       global.Config.Rustdesk.Key,
		},
	)
}
