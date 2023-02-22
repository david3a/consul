package gateways

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/hashicorp/consul/api"
	libassert "github.com/hashicorp/consul/test/integration/consul-container/libs/assert"
	libservice "github.com/hashicorp/consul/test/integration/consul-container/libs/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"math/rand"
	"testing"
	"time"
)

func getNamespace() string {
	return ""
}

// randomName generates a random name of n length with the provided
// prefix. If prefix is omitted, the then entire name is random char.
func randomName(prefix string, n int) string {
	if n == 0 {
		n = 32
	}
	if len(prefix) >= n {
		return prefix
	}
	rand.Seed(time.Now().UnixNano())
	p := make([]byte, n)
	rand.Read(p)
	return fmt.Sprintf("%s-%s", prefix, hex.EncodeToString(p))[:n]
}

func TestHTTPRouteFlattening(t *testing.T) {
	if testing.Short() {
		t.Skip("too slow for testing.Short")
	}

	//infrastructure set up
	listenerPort := 6000
	//create cluster
	cluster := createCluster(t, listenerPort)
	client := cluster.Agents[0].GetClient()
	service1Port := 8080
	service2Port := 8081
	serviceOne := createService(t, cluster, &libservice.ServiceOpts{
		Name:     "service1",
		ID:       "service1",
		HTTPPort: service1Port,
		GRPCPort: 8079,
	}, nil)
	serviceTwo := createService(t, cluster, &libservice.ServiceOpts{
		Name:     "service2",
		ID:       "service2",
		HTTPPort: service2Port,
		GRPCPort: 8082,
	}, []string{
		"-echo-debug-path", "/check",
	},
	)

	//TODO this should only matter in consul enterprise I believe?
	namespace := getNamespace()
	gatewayName := randomName("gw", 16)
	routeOneName := randomName("route", 16)
	routeTwoName := randomName("route", 16)
	path1 := "/"
	path2 := "/v2"

	//write config entries
	proxyDefaults := &api.ProxyConfigEntry{
		Kind:      api.ProxyDefaults,
		Name:      api.ProxyConfigGlobal,
		Namespace: namespace,
		Config: map[string]interface{}{
			"protocol": "http",
		},
	}

	_, _, err := client.ConfigEntries().Set(proxyDefaults, nil)
	assert.NoError(t, err)

	apiGateway := &api.APIGatewayConfigEntry{
		Kind: "api-gateway",
		Name: gatewayName,
		Listeners: []api.APIGatewayListener{
			{
				Port:     listenerPort,
				Protocol: "http",
				Hostname: "test.foo",
			},
		},
	}

	routeOne := &api.HTTPRouteConfigEntry{
		Kind: api.HTTPRoute,
		Name: routeOneName,
		Parents: []api.ResourceReference{
			{
				Kind:      api.APIGateway,
				Name:      gatewayName,
				Namespace: namespace,
			},
		},
		Hostnames: []string{
			"test.foo",
			"test.example",
		},
		Namespace: namespace,
		Rules: []api.HTTPRouteRule{
			{
				Services: []api.HTTPService{
					{
						Name:      serviceOne.GetServiceName(),
						Namespace: namespace,
					},
				},
				Matches: []api.HTTPMatch{
					{
						Path: api.HTTPPathMatch{
							Match: api.HTTPPathMatchPrefix,
							Value: path1,
						},
					},
				},
			},
		},
	}

	fmt.Println(path2)

	routeTwo := &api.HTTPRouteConfigEntry{
		Kind: api.HTTPRoute,
		Name: routeTwoName,
		Parents: []api.ResourceReference{
			{
				Kind:      api.APIGateway,
				Name:      gatewayName,
				Namespace: namespace,
			},
		},
		Hostnames: []string{
			"test.foo",
		},
		Namespace: namespace,
		Rules: []api.HTTPRouteRule{
			{
				Services: []api.HTTPService{
					{
						Name:      serviceTwo.GetServiceName(),
						Namespace: namespace,
					},
				},
				Matches: []api.HTTPMatch{
					{
						Path: api.HTTPPathMatch{
							Match: api.HTTPPathMatchPrefix,
							Value: path2,
						},
					},
					{
						Headers: []api.HTTPHeaderMatch{{
							Match: api.HTTPHeaderMatchExact,
							Name:  "x-v2",
							Value: "v2",
						}},
					},
				},
			},
		},
	}

	_, _, err = client.ConfigEntries().Set(apiGateway, nil)
	assert.NoError(t, err)
	_, _, err = client.ConfigEntries().Set(routeOne, nil)
	assert.NoError(t, err)
	_, _, err = client.ConfigEntries().Set(routeTwo, nil)
	assert.NoError(t, err)

	//create gateway service
	gatewayService, err := libservice.NewGatewayService(context.Background(), gatewayName, "api", cluster.Agents[0], listenerPort)
	require.NoError(t, err)
	libassert.CatalogServiceExists(t, client, gatewayName)

	//make sure config entries have been properly created
	require.Eventually(t, func() bool {
		entry, _, err := client.ConfigEntries().Get(api.APIGateway, gatewayName, &api.QueryOptions{Namespace: namespace})
		assert.NoError(t, err)
		if entry == nil {
			return false
		}
		apiEntry := entry.(*api.APIGatewayConfigEntry)
		t.Log(entry)
		return isAccepted(apiEntry.Status.Conditions)
	}, time.Second*10, time.Second*1)

	require.Eventually(t, func() bool {
		entry, _, err := client.ConfigEntries().Get(api.HTTPRoute, routeOneName, &api.QueryOptions{Namespace: namespace})
		assert.NoError(t, err)
		if entry == nil {
			return false
		}

		apiEntry := entry.(*api.HTTPRouteConfigEntry)
		t.Log(entry)
		return isBound(apiEntry.Status.Conditions)
	}, time.Second*10, time.Second*1)

	require.Eventually(t, func() bool {
		entry, _, err := client.ConfigEntries().Get(api.HTTPRoute, routeTwoName, nil)
		assert.NoError(t, err)
		if entry == nil {
			return false
		}

		apiEntry := entry.(*api.HTTPRouteConfigEntry)
		return isBound(apiEntry.Status.Conditions)
	}, time.Second*10, time.Second*1)

	//gateway resolves routes

	ip := "localhost"

	//route 2 with headers

	time.Sleep(time.Minute * 30)
	//Same path with and without header
	checkRoute(t, ip, gatewayService.GetPort(listenerPort), "v2/check", map[string]string{
		"Host": "test.foo",
		"x-v2": "v2",
	}, checkOptions{debug: true, statusCode: 200})

	checkRoute(t, ip, gatewayService.GetPort(listenerPort), "v2/check", map[string]string{
		"Host": "test.foo",
	}, checkOptions{statusCode: 200})

	checkRoute(t, ip, gatewayService.GetPort(listenerPort), "check", map[string]string{
		"Host": "test.foo",
		"x-v2": "v2",
	}, checkOptions{debug: true, statusCode: 200})

	checkRoute(t, ip, gatewayService.GetPort(listenerPort), "hello/hello", map[string]string{
		"Host": "test.foo",
		"x-v2": "v2",
	}, checkOptions{statusCode: 200})

	checkRoute(t, ip, gatewayService.GetPort(listenerPort), "v2", map[string]string{
		"Host": "test.foo",
		"x-v2": "v2",
	}, checkOptions{statusCode: 200})

	////hit service 1 by hitting path
	//checkRoute(t, ip, gatewayService.GetPort(listenerPort), "", map[string]string{
	//	"Host": "test.foo",
	//}, checkOptions{debug: false, statusCode: 200})

	//hit service 1 by hitting v2 path with v1 hostname
	checkRoute(t, ip, gatewayService.GetPort(listenerPort), "v2/check", map[string]string{
		"Host": "test.example",
	}, checkOptions{debug: false, statusCode: 200})

}
