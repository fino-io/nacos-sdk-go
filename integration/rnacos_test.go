package integration_test

import (
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/fino-io/nacos-sdk-go/v2/clients"
	"github.com/fino-io/nacos-sdk-go/v2/common/constant"
	"github.com/fino-io/nacos-sdk-go/v2/model"
	"github.com/fino-io/nacos-sdk-go/v2/vo"
	"github.com/stretchr/testify/require"
)

// TestRNacos is opt-in: credentials and server address must be supplied by the caller.
func TestRNacos(t *testing.T) {
	host := os.Getenv("NACOS_TEST_HOST")
	if host == "" {
		t.Skip("set NACOS_TEST_HOST, NACOS_TEST_PORT, NACOS_TEST_USERNAME and NACOS_TEST_PASSWORD")
	}
	port, err := strconv.ParseUint(os.Getenv("NACOS_TEST_PORT"), 10, 64)
	require.NoError(t, err)
	require.NotEmpty(t, os.Getenv("NACOS_TEST_PASSWORD"))
	root := t.TempDir()
	param := vo.NacosClientParam{
		ClientConfig: &constant.ClientConfig{
			UpdateCacheWhenEmpty: true, TimeoutMs: 5000, Username: os.Getenv("NACOS_TEST_USERNAME"), Password: os.Getenv("NACOS_TEST_PASSWORD"),
			LogDir: root + "/logs", CacheDir: root + "/cache", LogLevel: "warn", NotLoadCacheAtStart: true, DisableUseSnapShot: true,
		},
		ServerConfigs: []constant.ServerConfig{{IpAddr: host, Port: port}},
	}
	config, err := clients.NewConfigClient(param)
	require.NoError(t, err)
	defer config.CloseClient()
	naming, err := clients.NewNamingClient(param)
	require.NoError(t, err)
	defer naming.CloseClient()
	id := fmt.Sprintf("sdk-smoke-%d", time.Now().UnixNano())
	const group = "SDK_SMOKE_TEST"
	cp := vo.ConfigParam{DataId: id, Group: group, Content: "version=1"}
	defer func() { ok, err := config.DeleteConfig(cp); require.NoError(t, err); require.True(t, ok) }()
	ok, err := config.PublishConfig(cp)
	require.NoError(t, err)
	require.True(t, ok)
	content, err := config.GetConfig(cp)
	require.NoError(t, err)
	require.Equal(t, cp.Content, content)
	t.Log("authenticated config publish/read passed")
	changes := make(chan string, 16)
	listen := vo.ConfigParam{DataId: id, Group: group, OnChange: func(_, _, _, content string) {
		select {
		case changes <- content:
		default:
		}
	}}
	require.NoError(t, config.ListenConfig(listen))
	defer func() { require.NoError(t, config.CancelListenConfig(listen)) }()
	cp.Content = "version=2"
	ok, err = config.PublishConfig(cp)
	require.NoError(t, err)
	require.True(t, ok)
	deadline := time.After(30 * time.Second)
waitConfig:
	for {
		select {
		case value := <-changes:
			if value == cp.Content {
				break waitConfig
			}
		case <-deadline:
			t.Fatal("config change notification timed out")
		}
	}
	t.Log("config change push passed")
	updates := make(chan []model.Instance, 16)
	sub := &vo.SubscribeParam{ServiceName: id, GroupName: group, SubscribeCallback: func(instances []model.Instance, err error) {
		if err == nil {
			select {
			case updates <- instances:
			default:
			}
		}
	}}
	require.NoError(t, naming.Subscribe(sub))
	defer func() { require.NoError(t, naming.Unsubscribe(sub)) }()
	instance := vo.RegisterInstanceParam{Ip: "127.0.0.1", Port: 18080, Weight: 1, Enable: true, Healthy: true, Ephemeral: true, ServiceName: id, GroupName: group, ClusterName: "DEFAULT"}
	dereg := vo.DeregisterInstanceParam{Ip: instance.Ip, Port: instance.Port, ServiceName: id, GroupName: group, Cluster: "DEFAULT", Ephemeral: true}
	defer func() { ok, err := naming.DeregisterInstance(dereg); require.NoError(t, err); require.True(t, ok) }()
	ok, err = naming.RegisterInstance(instance)
	require.NoError(t, err)
	require.True(t, ok)
	require.Eventually(t, func() bool {
		found, err := naming.SelectOneHealthyInstance(vo.SelectOneHealthInstanceParam{ServiceName: id, GroupName: group})
		return err == nil && found != nil && found.Ip == instance.Ip && found.Port == instance.Port
	}, 20*time.Second, 200*time.Millisecond)
	t.Log("service registration/discovery passed")
	deadline = time.After(20 * time.Second)
waitService:
	for {
		select {
		case instances := <-updates:
			for _, found := range instances {
				if found.Ip == instance.Ip && found.Port == instance.Port {
					break waitService
				}
			}
		case <-deadline:
			t.Fatal("service subscription notification timed out")
		}
	}
	t.Log("service subscription push passed")
	// A second client reads from the server with a separate empty local cache.
	param.ClientConfig.CacheDir = root + "/fresh-cache"
	fresh, err := clients.NewConfigClient(param)
	require.NoError(t, err)
	defer fresh.CloseClient()
	content, err = fresh.GetConfig(cp)
	require.NoError(t, err)
	require.Equal(t, "version=2", content)
	t.Log("fresh client config recovery passed")
	ok, err = naming.DeregisterInstance(dereg)
	require.NoError(t, err)
	require.True(t, ok)
	require.Eventually(t, func() bool {
		found, err := naming.SelectAllInstances(vo.SelectAllInstancesParam{ServiceName: id, GroupName: group})
		return err == nil && len(found) == 0
	}, 20*time.Second, 200*time.Millisecond)
	t.Log("service deregistration passed")
}
