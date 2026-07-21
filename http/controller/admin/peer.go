package admin

import (
	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/global"
	"github.com/lejianwen/rustdesk-api/v2/http/request/admin"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/service"
	"gorm.io/gorm"
	"strconv"
	"time"
)

type Peer struct {
}

// Detail 设备
// @Tags 设备
// @Summary 设备详情
// @Description 设备详情
// @Accept  json
// @Produce  json
// @Param id path int true "ID"
// @Success 200 {object} response.Response{data=model.Peer}
// @Failure 500 {object} response.Response
// @Router /admin/peer/detail/{id} [get]
// @Security token
func (ct *Peer) Detail(c *gin.Context) {
	id := c.Param("id")
	iid, _ := strconv.Atoi(id)
	u := service.AllService.PeerService.InfoByRowId(uint(iid))
	if u.RowId > 0 {
		response.Success(c, u)
		return
	}
	response.Fail(c, 101, response.TranslateMsg(c, "ItemNotFound"))
	return
}

// Create 创建设备
// @Tags 设备
// @Summary 创建设备
// @Description 创建设备
// @Accept  json
// @Produce  json
// @Param body body admin.PeerForm true "设备信息"
// @Success 200 {object} response.Response{data=model.Peer}
// @Failure 500 {object} response.Response
// @Router /admin/peer/create [post]
// @Security token
func (ct *Peer) Create(c *gin.Context) {
	f := &admin.PeerForm{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	errList := global.Validator.ValidStruct(c, f)
	if len(errList) > 0 {
		response.Fail(c, 101, errList[0])
		return
	}
	p := f.ToPeer()
	// P3-1 设备审批：管理员手动创建的设备视为可信，Status 保持零值，
	// gorm 会按 default:1 标签写入"已通过"，无需额外处理。
	err := service.AllService.PeerService.Create(p)
	if err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}
	response.Success(c, nil)
}

// List 列表
// @Tags 设备
// @Summary 设备列表
// @Description 设备列表
// @Accept  json
// @Produce  json
// @Param page query int false "页码"
// @Param page_size query int false "页大小"
// @Param time_ago query int false "时间"
// @Param id query string false "ID"
// @Param hostname query string false "主机名"
// @Param uuids query string false "uuids 用逗号分隔"
// @Success 200 {object} response.Response{data=model.PeerList}
// @Failure 500 {object} response.Response
// @Router /admin/peer/list [get]
// @Security token
func (ct *Peer) List(c *gin.Context) {
	query := &admin.PeerQuery{}
	if err := c.ShouldBindQuery(query); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	res := service.AllService.PeerService.List(query.Page, query.PageSize, func(tx *gorm.DB) {
		if query.TimeAgo > 0 {
			lt := time.Now().Unix() - int64(query.TimeAgo)
			tx.Where("last_online_time < ?", lt)
		}
		if query.TimeAgo < 0 {
			lt := time.Now().Unix() + int64(query.TimeAgo)
			tx.Where("last_online_time > ?", lt)
		}
		if query.Id != "" {
			tx.Where("id like ?", "%"+query.Id+"%")
		}
		if query.Hostname != "" {
			tx.Where("hostname like ?", "%"+query.Hostname+"%")
		}
		if query.Uuids != "" {
			tx.Where("uuid in (?)", query.Uuids)
		}
		if query.Username != "" {
			tx.Where("username like ?", "%"+query.Username+"%")
		}
		if query.Ip != "" {
			tx.Where("last_online_ip like ?", "%"+query.Ip+"%")
		}
		if query.Alias != "" {
			tx.Where("alias like ?", "%"+query.Alias+"%")
		}
		// P3-1 设备审批：按审批状态过滤（0 待审批 / 1 已通过），未传参则不过滤
		if query.Status != nil {
			tx.Where("status = ?", *query.Status)
		}
	})
	response.Success(c, res)
}

// Update 编辑
// @Tags 设备
// @Summary 设备编辑
// @Description 设备编辑
// @Accept  json
// @Produce  json
// @Param body body admin.PeerForm true "设备信息"
// @Success 200 {object} response.Response{data=model.Peer}
// @Failure 500 {object} response.Response
// @Router /admin/peer/update [post]
// @Security token
func (ct *Peer) Update(c *gin.Context) {
	f := &admin.PeerForm{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	if f.RowId == 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError"))
		return
	}
	errList := global.Validator.ValidStruct(c, f)
	if len(errList) > 0 {
		response.Fail(c, 101, errList[0])
		return
	}
	u := f.ToPeer()
	err := service.AllService.PeerService.Update(u)
	if err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}
	response.Success(c, nil)
}

// Delete 删除
// @Tags 设备
// @Summary 设备删除
// @Description 设备删除
// @Accept  json
// @Produce  json
// @Param body body admin.PeerForm true "设备信息"
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/peer/delete [post]
// @Security token
func (ct *Peer) Delete(c *gin.Context) {
	f := &admin.PeerForm{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	id := f.RowId
	errList := global.Validator.ValidVar(c, id, "required,gt=0")
	if len(errList) > 0 {
		response.Fail(c, 101, errList[0])
		return
	}
	u := service.AllService.PeerService.InfoByRowId(f.RowId)
	if u.RowId > 0 {
		err := service.AllService.PeerService.Delete(u)
		if err == nil {
			response.Success(c, nil)
			return
		}
		response.Fail(c, 101, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}
	response.Fail(c, 101, response.TranslateMsg(c, "ItemNotFound"))
}

// BatchDelete 批量删除
// @Tags 设备
// @Summary 批量设备删除
// @Description 批量设备删除
// @Accept  json
// @Produce  json
// @Param body body admin.PeerBatchDeleteForm true "设备id"
// @Success 200 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /admin/peer/batchDelete [post]
// @Security token
func (ct *Peer) BatchDelete(c *gin.Context) {
	f := &admin.PeerBatchDeleteForm{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	if len(f.RowIds) == 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError"))
		return
	}
	err := service.AllService.PeerService.BatchDelete(f.RowIds)
	if err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}
	response.Success(c, nil)
}

func (ct *Peer) SimpleData(c *gin.Context) {
	f := &admin.SimpleDataQuery{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	if len(f.Ids) == 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError"))
		return
	}
	res := service.AllService.PeerService.List(1, 99999, func(tx *gorm.DB) {
		//可以公开的情报
		tx.Select("id,version")
		tx.Where("id in (?)", f.Ids)
	})
	response.Success(c, res)
}

// Approve 设备审批通过
// @Tags 设备
// @Summary 设备审批通过（批量）
// @Description 将指定设备的审批状态置为"已通过"（P3-1 设备审批），返回实际更新条数
// @Accept  json
// @Produce  json
// @Param body body admin.PeerBatchApproveForm true "设备 row_id 列表"
// @Success 200 {object} response.Response{data=gin.H}
// @Failure 500 {object} response.Response
// @Router /admin/peer/approve [post]
// @Security token
func (ct *Peer) Approve(c *gin.Context) {
	f := &admin.PeerBatchApproveForm{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	if len(f.RowIds) == 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError"))
		return
	}
	affected, err := service.AllService.PeerService.BatchApprove(f.RowIds)
	if err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}
	response.Success(c, &gin.H{"affected": affected})
}

// BatchUpdateTags 按设备批量更新标签
// @Tags 设备
// @Summary 按设备批量更新标签（管理员粒度）
// @Description 入参 row_ids 为设备 id（peers.row_id）列表，作用于这些设备对应的全部地址簿条目
// @Description （关联关系：address_book.id = peers.id，跨用户条目一并更新）；tags 为全量替换后的
// @Description 标签数组（与既有 tag 模型一致：address_book.tags 存 JSON 数组）。
// @Description 未加入任何地址簿的设备会被静默跳过，不报错；data.affected 返回实际更新的条目数。
// @Accept  json
// @Produce  json
// @Param body body admin.BatchUpdateTagsForm true "设备 peers.row_id 列表 + 标签数组"
// @Success 200 {object} response.Response{data=gin.H}
// @Failure 500 {object} response.Response
// @Router /admin/peer/batchUpdateTags [post]
// @Security token
func (ct *Peer) BatchUpdateTags(c *gin.Context) {
	f := &admin.BatchUpdateTagsForm{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	if len(f.RowIds) == 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError"))
		return
	}
	// row_ids 为 peers.row_id（前端设备管理页选中的是设备）：先取设备标识 peers.id，
	// 地址簿条目通过 address_book.id = peers.id 与设备关联。
	peerIds, err := service.AllService.PeerService.GetIdListByRowIds(f.RowIds)
	if err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "OperationFailed")+err.Error())
		return
	}
	var affected int64 = 0
	if len(peerIds) > 0 {
		// 管理员粒度：不按 user_id 过滤，跨用户条目一并更新；
		// 不在任何地址簿中的设备匹配不到条目，被静默跳过（affected 不计）。
		affected, err = service.AllService.AddressBookService.BatchUpdateTagsByPeerIds(peerIds, f.Tags)
		if err != nil {
			response.Fail(c, 101, response.TranslateMsg(c, "OperationFailed")+err.Error())
			return
		}
	}
	response.Success(c, &gin.H{"affected": affected})
}
