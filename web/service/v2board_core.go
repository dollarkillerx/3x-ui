package service

import (
	"encoding/json"
	"fmt"
	"log"
	"resty.dev/v3"
	"time"
	"x-ui/database"
	"x-ui/database/model"
	"x-ui/web/v2board"
	"x-ui/xray"
)

type V2boardCore struct {
	xrayApi xray.XrayAPI
}

func NewV2boardCore() *V2boardCore {
	return &V2boardCore{}
}

func (c *V2boardCore) Run() {
	config := v2board.GetV2boardConfig()
	pollTicker := time.NewTicker(time.Second * time.Duration(config.NodePollingInterval))
	pushTicker := time.NewTicker(time.Second * time.Duration(config.NodePushingInterval))

	c.syncUsers() // 首次同步用户数据
	for {
		select {
		case <-pollTicker.C:
			go c.syncUsers()
		case <-pushTicker.C:
			go c.reportTraffic()
		}
	}
}

// SyncUsers 同步用户数据
func (c *V2boardCore) syncUsers() {
	// 实现用户从面板拉取并写入本地配置等逻辑
	config := v2board.GetV2boardConfig()
	client := resty.New()
	defer client.Close()

	// ?node_id=6&node_type=vless&token=f0f056fb-5428-4ffc-8dd2-7c091962de7f
	resp, err := client.R().
		SetQueryParams(map[string]string{
			"token":     config.ApiKey,
			"node_type": config.NodeType,
			"node_id":   config.NodeID,
		}).
		Get(fmt.Sprintf("%s/api/v1/server/UniProxy/user", config.ApiHost))

	if err != nil {
		log.Printf("Error syncing users: %s \n", err)
		return
	}

	var users v2board.Users
	err = json.Unmarshal(resp.Bytes(), &users)
	if err != nil {
		log.Printf("Error syncing users: %s \n", err)
		return
	}

	oldInbound, err := c.GetInbound(1)
	if err != nil {
		log.Printf("Error syncing users: %s \n", err)
		return
	}

	// 更新当前节点用户数据
	c.xrayApi.Init(p.GetAPIPort())
	defer c.xrayApi.Close()
	for _, user := range users.Users {
		err := c.xrayApi.AddUser(string(oldInbound.Protocol), oldInbound.Tag, map[string]any{
			"email":    user.Uuid,
			"id":       user.Uuid,
			"security": "",
			"flow":     "",
			"password": user.Uuid,
			"cipher":   "",
		})

		if err != nil {
			log.Printf("Error syncing users: %s \n", err)
			return
		}
	}

	// 删除用户
	inboundUsers, err := c.xrayApi.GetInboundUsers(oldInbound.Tag)
	if err != nil {
		log.Printf("Error syncing users: %s \n", err)
		return
	}

	for _, inboundUser := range inboundUsers {
		found := false
		for _, user := range users.Users {
			if inboundUser.Email == user.Uuid { // 假设 user.Uuid 是面板中用于匹配的标识
				found = true
				break
			}
		}

		if !found {
			err := c.xrayApi.RemoveUser(oldInbound.Tag, inboundUser.Email)
			if err != nil {
				log.Printf("无法删除用户 %s: %v \n", inboundUser.Email, err)
			} else {
				log.Printf("已删除过期用户: %s \n", inboundUser.Email)
			}
		}
	}

}

// ReportTraffic 上报用户流量
func (c *V2boardCore) reportTraffic() {
	// 实现向面板提交每个用户的流量统计

}

func (c *V2boardCore) GetInbound(id int) (*model.Inbound, error) {
	db := database.GetDB()
	inbound := &model.Inbound{}
	err := db.Model(model.Inbound{}).First(inbound, id).Error
	if err != nil {
		return nil, err
	}
	return inbound, nil
}
