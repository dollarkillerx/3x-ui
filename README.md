
- /panel/inbound/addClient  POST 添加新用户
- /panel/inbound/list POST 入口&用户列表

``` 
//s.xrayApi.AddUser()
//s.xrayApi.RemoveUser()

traffic, clientTraffics, err := s.xrayApi.GetTraffic(true)
if err != nil {
    // traffic 总流量
    // clientTraffics 每个客户端的流量
}

// web/service/inbound.go
```

docker build -t dollarkiller/x3pro:latest .
docker push dollarkiller/x3pro:latest
