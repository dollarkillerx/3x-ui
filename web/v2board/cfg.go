package v2board

import (
	"os"
	"strconv"
)

type UserItem struct {
	Id         int         `json:"id"`
	Uuid       string      `json:"uuid"`
	SpeedLimit interface{} `json:"speed_limit"`
}

type Users struct {
	Users []UserItem `json:"users"`
}

type V2boardConfig struct {
	ApiHost             string `json:"ApiHost"`             // V2board API Host
	ApiKey              string `json:"ApiKey"`              // V2board API Key
	NodeID              string `json:"NodeId"`              // V2board Node ID
	NodeType            string `json:"NodeType"`            // V2board Node Type default Shadowsocks
	NodePollingInterval int    `json:"NodePollingInterval"` // V2board Node Polling Interval
	NodePushingInterval int    `json:"NodePushingInterval"` // V2board Node Pushing Interval
}

func GetV2boardConfig() V2boardConfig {
	return V2boardConfig{
		ApiHost:             getEnv("API_HOST", "https://api.xxx.org"),
		ApiKey:              getEnv("API_KEY", "ae97-a208e1ed4f09"),
		NodeID:              getEnv("NODE_ID", "5"),
		NodeType:            getEnv("NODE_TYPE", "shadowsocks"),
		NodePollingInterval: getEnvAsInt("NODE_POLLING_INTERVAL", 60),
		NodePushingInterval: getEnvAsInt("NODE_PUSHING_INTERVAL", 60),
	}
}

// 获取字符串环境变量，有默认值
func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return defaultVal
}

// 获取 int 类型环境变量，有默认值
func getEnvAsInt(key string, defaultVal int) int {
	if valStr, ok := os.LookupEnv(key); ok {
		if val, err := strconv.Atoi(valStr); err == nil {
			return val
		}
	}
	return defaultVal
}
