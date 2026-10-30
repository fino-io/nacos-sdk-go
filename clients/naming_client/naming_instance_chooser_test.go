package naming_client

import (
	"fmt"
	"math"
	"testing"

	"github.com/fino-io/nacos-sdk-go/v2/model"
	"github.com/stretchr/testify/require"
)

func TestHardeningWeightedSelection(t *testing.T) {
	for _, tc := range []struct {
		name    string
		weights []float64
		ratio   float64
	}{
		{"fractional", []float64{0.2, 0.8}, .2},
		{"mixed", []float64{0.5, 1.5}, .25},
		{"integer", []float64{1, 3}, .25},
		{"very_large", []float64{math.MaxFloat64, math.MaxFloat64}, .5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			service := model.Service{Hosts: []model.Instance{
				{Ip: "first", Weight: tc.weights[0], Healthy: true, Enable: true},
				{Ip: "second", Weight: tc.weights[1], Healthy: true, Enable: true},
			}}
			first := 0
			require.NotPanics(t, func() {
				for i := 0; i < 10000; i++ {
					found, err := (&NamingClient{}).selectOneHealthyInstances(service)
					require.NoError(t, err)
					if found.Ip == "first" {
						first++
					}
				}
			})
			require.InDelta(t, tc.ratio, float64(first)/10000, .04)
			require.Equal(t, "first", service.Hosts[0].Ip, "selection must not reorder cached hosts")
		})
	}
}

func TestHardeningInvalidWeights(t *testing.T) {
	for _, weight := range []float64{0, -1, math.NaN(), math.Inf(1), math.Inf(-1)} {
		t.Run(fmt.Sprint(weight), func(t *testing.T) {
			service := model.Service{Hosts: []model.Instance{{Weight: weight, Healthy: true, Enable: true}}}
			require.NotPanics(t, func() {
				found, err := (&NamingClient{}).selectOneHealthyInstances(service)
				require.Error(t, err)
				require.Nil(t, found)
			})
		})
	}
	service := model.Service{Hosts: []model.Instance{
		{Ip: "disabled", Weight: 10, Healthy: true}, {Ip: "unhealthy", Weight: 10, Enable: true},
		{Ip: "valid", Weight: .1, Healthy: true, Enable: true},
	}}
	require.NotPanics(t, func() {
		found, err := (&NamingClient{}).selectOneHealthyInstances(service)
		require.NoError(t, err)
		require.Equal(t, "valid", found.Ip)
	})
}
