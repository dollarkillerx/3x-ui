package v2board

import (
	"encoding/json"
	"fmt"
	"log"
	"resty.dev/v3"
	"testing"
)

func TestGetV2boardConfig(t *testing.T) {
	config := GetV2boardConfig()
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

	var users Users
	err = json.Unmarshal(resp.Bytes(), &users)
	if err != nil {
		log.Printf("Error syncing users: %s \n", err)
		return
	}

	fmt.Println(len(users.Users))
}
