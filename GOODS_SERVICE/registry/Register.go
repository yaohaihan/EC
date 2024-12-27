package registry

import "github.com/hashicorp/consul/api"

// Register 自定义一个注册中心的抽象
type Register interface {
	// 注册
	RegisterService(serviceName string, ip string, port int, tags []string) error
	// 服务发现
	ListService(serviceName string) (map[string]*api.AgentService, error)
	// 注销
	Deregister(serviceID string) error
}
