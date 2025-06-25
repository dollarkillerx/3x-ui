package service

import (
	"encoding/json"
	"fmt"
	"golang.org/x/sync/errgroup"
	"log"
	"resty.dev/v3"
	"strconv"
	"sync"
	"time"
	"x-ui/database"
	"x-ui/database/model"
	"x-ui/logger"
	"x-ui/web/v2board"
	"x-ui/xray"
)

type V2boardCore struct {
	xrayApi xray.XrayAPI
}

func NewV2boardCore() *V2boardCore {
	return &V2boardCore{
		xrayApi: xray.XrayAPI{},
	}
}

func (c *V2boardCore) Run() {
	//config := v2board.GetV2boardConfig()
	//pollTicker := time.NewTicker(time.Second * time.Duration(config.NodePollingInterval))
	//pushTicker := time.NewTicker(time.Second * time.Duration(config.NodePushingInterval))

	pollTicker := time.NewTicker(time.Second * 60)
	pushTicker := time.NewTicker(time.Second * 60)

	c.syncUsers(true) // 首次同步用户数据
	for {
		select {
		case <-pollTicker.C:
			go c.syncUsers(false)
		case <-pushTicker.C:
			go c.reportTraffic()
		}
	}
}

var (
	localUserCache = make(map[string]v2board.UserItem)
	userCacheLock  sync.Mutex
)

// SyncUsers 同步用户数据（支持初始化模式）
func (c *V2boardCore) syncUsers(initial bool) {
	fmt.Println("用户数据同步中.........")
	config := v2board.GetV2boardConfig()
	client := resty.New()
	defer client.Close()

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

	var remoteUsers v2board.Users
	err = json.Unmarshal(resp.Bytes(), &remoteUsers)
	if err != nil {
		log.Printf("Error unmarshaling users: %s \n", err)
		return
	}

	fmt.Printf("获得用户数据: %d 个\n", len(remoteUsers.Users))

	oldInbound, err := c.GetInbound("v2board")
	if err != nil {
		log.Printf("Error getting inbound: %s \n", err)
		return
	}

	err = c.xrayApi.Init(p.GetAPIPort())
	if err != nil {
		log.Printf("Xray API init failed: %s \n", err)
		return
	}
	defer c.xrayApi.Close()

	// 如果是初始化，先删除所有老用户
	inboundUsers, _ := c.xrayApi.GetInboundUsers(oldInbound.Tag)
	fmt.Println("当前系统用户数: ", len(inboundUsers))
	if initial {
		for _, u := range inboundUsers {
			if u.Email == "admin" {
				continue
			}
			_ = c.xrayApi.RemoveUser2(oldInbound.Tag, u.Email)
		}
	}

	var oldSettings map[string]any
	err = json.Unmarshal([]byte(oldInbound.Settings), &oldSettings)
	if err != nil {
		log.Printf("Error unmarshaling settings: %s \n", err)
		return
	}

	// 去重 remoteUsers.Users
	uniqueUsers := make([]v2board.UserItem, 0)
	seen := make(map[int]struct{}) // 如果是按 Uuid 去重可以改为 map[string]struct{}
	for _, user := range remoteUsers.Users {
		if _, ok := seen[user.Id]; !ok {
			seen[user.Id] = struct{}{}
			uniqueUsers = append(uniqueUsers, user)
		}
	}

	// 并发添加用户，最多10个并发
	semaphore := make(chan struct{}, 10)
	var g errgroup.Group

	for _, user := range uniqueUsers {
		user := user // 避免闭包捕获问题
		semaphore <- struct{}{}
		g.Go(func() error {
			defer func() { <-semaphore }()

			email := strconv.Itoa(user.Id)

			userCacheLock.Lock()
			localUser, exists := localUserCache[email]
			if exists && localUser.Uuid != user.Uuid {
				_ = c.xrayApi.RemoveUser2(oldInbound.Tag, email)
				exists = false
			}
			userCacheLock.Unlock()

			if !exists {
				cipher := ""
				if oldInbound.Protocol == "shadowsocks" {
					cipher = oldSettings["method"].(string)
				}
				err := c.xrayApi.AddUser(string(oldInbound.Protocol), oldInbound.Tag, map[string]any{
					"email":    email,
					"id":       user.Uuid,
					"password": user.Uuid,
					"cipher":   cipher,
					"security": "",
					"flow":     "",
				})
				if err != nil {
					log.Printf("Xray API add user failed: %s \n", err)
					return nil
				}

				userCacheLock.Lock()
				localUserCache[email] = user
				userCacheLock.Unlock()
			}
			return nil
		})
	}

	// 等待所有任务完成
	_ = g.Wait()

	if !initial {
		// 差异对比：删除远程不存在的用户
		inboundUsers, _ := c.xrayApi.GetInboundUsers(oldInbound.Tag)
		remoteMap := make(map[string]struct{})
		for _, u := range remoteUsers.Users {
			remoteMap[strconv.Itoa(u.Id)] = struct{}{}
		}
		userCacheLock.Lock()
		for _, inboundUser := range inboundUsers {
			if inboundUser.Email == "admin" {
				continue
			}
			if _, ok := remoteMap[inboundUser.Email]; !ok {
				_ = c.xrayApi.RemoveUser2(oldInbound.Tag, inboundUser.Email)
				fmt.Println("删除用户:", inboundUser.Email)
				delete(localUserCache, inboundUser.Email)
			}
		}
		userCacheLock.Unlock()
	}

	fmt.Println("用户数据同步完成")
}

// ReportTraffic 上报用户流量
func (c *V2boardCore) reportTraffic() {
	fmt.Println("上报用户流量中.........")
	c.xrayApi.Init(p.GetAPIPort())
	defer c.xrayApi.Close()

	// 实现向面板提交每个用户的流量统计
	_, clientTraffic, err := c.xrayApi.GetTraffic(true)
	if err != nil {
		logger.Debug("Failed to fetch Xray traffic:", err)
		return
	}

	client := resty.New()
	defer client.Close()

	// ?node_id=6&node_type=vless&token=f0f056fb-5428-4ffc-8dd2-7c091962de7f
	config := v2board.GetV2boardConfig()

	var trafficData map[int][]int64 // 用户id: []int64{上行流量,下行流量}
	trafficData = make(map[int][]int64)
	for _, cx := range clientTraffic {
		if cx.Up == 0 && cx.Down == 0 {
			continue
		}
		// 将流量数据提交给面板

		atoi, err := strconv.Atoi(cx.Email)
		if err != nil {
			continue
		}

		trafficData[atoi] = []int64{cx.Up, cx.Down}
	}

	if len(trafficData) == 0 {
		fmt.Println("无流量数据")
		return
	}
	resp, err := client.R().
		SetQueryParams(map[string]string{
			"token":     config.ApiKey,
			"node_type": config.NodeType,
			"node_id":   config.NodeID,
		}).
		SetHeader("Content-Type", "application/json").
		SetBody(trafficData).
		Post(fmt.Sprintf("%s/api/v1/server/UniProxy/push", config.ApiHost))
	if err != nil {
		log.Printf("Error syncing users: %s \n", err)
		return
	}

	if resp.StatusCode() != 200 {
		log.Printf("Error syncing users: %s \n", err)
		return
	}

	fmt.Printf("流量上报成功 %d \n", len(trafficData))
}

func (c *V2boardCore) GetInbound(id string) (*model.Inbound, error) {
	db := database.GetDB()
	inbound := &model.Inbound{}
	err := db.Model(model.Inbound{}).Where("remark = ?", id).First(inbound).Error
	if err != nil {
		return nil, err
	}
	return inbound, nil
}
