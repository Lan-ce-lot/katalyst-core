/*
Copyright 2022 The Katalyst Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package provisionpolicy

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8types "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"

	katalyst_base "github.com/kubewharf/katalyst-core/cmd/base"
	"github.com/kubewharf/katalyst-core/cmd/katalyst-agent/app/options"
	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/metacache"
	"github.com/kubewharf/katalyst-core/pkg/agent/sysadvisor/types"
	"github.com/kubewharf/katalyst-core/pkg/config"
	provisionconf "github.com/kubewharf/katalyst-core/pkg/config/agent/sysadvisor/qosaware/resource/cpu/provision"
	"github.com/kubewharf/katalyst-core/pkg/consts"
	"github.com/kubewharf/katalyst-core/pkg/metaserver"
	"github.com/kubewharf/katalyst-core/pkg/metaserver/agent/metric"
	"github.com/kubewharf/katalyst-core/pkg/metaserver/agent/pod"
	"github.com/kubewharf/katalyst-core/pkg/metrics"
	metricspool "github.com/kubewharf/katalyst-core/pkg/metrics/metrics-pool"
	"github.com/kubewharf/katalyst-core/pkg/util/machine"
)

var (
	metaCacheRama  *metacache.MetaCacheImp
	metaServerRama *metaserver.MetaServer
)

func generateRamaTestConfiguration(t *testing.T, checkpointDir, stateFileDir, checkpointManagerDir string) *config.Configuration {
	conf, err := options.NewOptions().Config()
	require.NoError(t, err)
	require.NotNil(t, conf)

	conf.GenericSysAdvisorConfiguration.StateFileDirectory = stateFileDir
	conf.MetaServerConfiguration.CheckpointManagerDir = checkpointDir
	conf.CheckpointManagerDir = checkpointManagerDir

	conf.RegionIndicatorTargetConfiguration = map[types.QoSRegionType][]provisionconf.IndicatorTargetConfiguration{
		types.QoSRegionTypeShare: {
			{
				Name: consts.MetricCPUSchedwait,
			},
		},
		types.QoSRegionTypeDedicatedNumaExclusive: {
			{
				Name: consts.MetricCPUCPIContainer,
			},
			{
				Name: consts.MetricMemBandwidthNuma,
			},
		},
	}

	conf.PolicyRama = &provisionconf.PolicyRamaConfiguration{
		PIDParameters: map[string]types.FirstOrderPIDParams{
			consts.MetricCPUSchedwait: {
				Kpp:                  10.0,
				Kpn:                  1.0,
				Kdp:                  0.0,
				Kdn:                  0.0,
				AdjustmentUpperBound: types.MaxRampUpStep,
				AdjustmentLowerBound: -types.MaxRampDownStep,
				DeadbandLowerPct:     0.8,
				DeadbandUpperPct:     0.05,
			},
			consts.MetricCPUCPIContainer: {
				Kpp:                  10.0,
				Kpn:                  1.0,
				Kdp:                  0.0,
				Kdn:                  0.0,
				AdjustmentUpperBound: types.MaxRampUpStep,
				AdjustmentLowerBound: -types.MaxRampDownStep,
				DeadbandLowerPct:     0.95,
				DeadbandUpperPct:     0.02,
			},
			consts.MetricMemBandwidthNuma: {
				Kpp:                  10.0,
				Kpn:                  1.0,
				Kdp:                  0.0,
				Kdn:                  0.0,
				AdjustmentUpperBound: types.MaxRampUpStep,
				AdjustmentLowerBound: -types.MaxRampDownStep,
				DeadbandLowerPct:     0.95,
				DeadbandUpperPct:     0.02,
			},
		},
	}

	conf.GetDynamicConfiguration().EnableReclaim = true

	return conf
}

func newTestPolicyRama(t *testing.T, checkpointDir string, stateFileDir string, checkpointManagerDir string, regionInfo types.RegionInfo, podSet types.PodSet) ProvisionPolicy {
	conf := generateRamaTestConfiguration(t, checkpointDir, stateFileDir, checkpointManagerDir)

	metaCacheTmp, err := metacache.NewMetaCacheImp(conf, metricspool.DummyMetricsEmitterPool{}, metric.NewFakeMetricsFetcher(metrics.DummyMetrics{}))
	metaCacheRama = metaCacheTmp
	require.NoError(t, err)
	require.NotNil(t, metaCacheRama)

	genericCtx, err := katalyst_base.GenerateFakeGenericContext([]runtime.Object{})
	require.NoError(t, err)

	metaServerTmp, err := metaserver.NewMetaServer(genericCtx.Client, metrics.DummyMetrics{}, conf)
	metaServerRama = metaServerTmp
	assert.NoError(t, err)
	require.NotNil(t, metaServerRama)

	p := NewPolicyRama(regionInfo.RegionName, regionInfo.RegionType, regionInfo.OwnerPoolName, conf, nil, metaCacheRama, metaServerRama, metrics.DummyMetrics{})
	metaCacheRama.SetRegionInfo(regionInfo.RegionName, &regionInfo)

	p.SetBindingNumas(regionInfo.BindingNumas)
	p.SetPodSet(podSet)

	return p
}

func constructPodFetcherRama(names []string) pod.PodFetcher {
	var pods []*v1.Pod
	for _, name := range names {
		pods = append(pods, &v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
				UID:  k8types.UID(name),
			},
		})
	}

	return &pod.PodFetcherStub{PodList: pods}
}

func TestPolicyRama(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name               string
		regionInfo         types.RegionInfo
		podSet             types.PodSet
		resourceEssentials types.ResourceEssentials
		controlEssentials  types.ControlEssentials
		wantResult         types.ControlKnob
	}{
		{
			name: "share_ramp_up",
			podSet: types.PodSet{
				"pod0": sets.String{
					"container0": struct{}{},
				},
			},
			regionInfo: types.RegionInfo{
				RegionName:   "share-xxx",
				RegionType:   types.QoSRegionTypeShare,
				BindingNumas: machine.NewCPUSet(0),
			},
			resourceEssentials: types.ResourceEssentials{
				EnableReclaim:       true,
				ResourceUpperBound:  90,
				ResourceLowerBound:  4,
				ReservedForAllocate: 0,
			},
			controlEssentials: types.ControlEssentials{
				ControlKnobs: types.ControlKnob{
					types.ControlKnobNonReclaimedCPUSize: {
						Value:  40,
						Action: types.ControlKnobActionNone,
					},
				},
				Indicators: types.Indicator{
					consts.MetricCPUSchedwait: {
						Current: 800,
						Target:  400,
					},
				},
				ReclaimOverlap: false,
			},
			wantResult: types.ControlKnob{
				types.ControlKnobNonReclaimedCPUSize: {
					Value:  48,
					Action: types.ControlKnobActionNone,
				},
			},
		},
		{
			name: "share_ramp_down",
			podSet: types.PodSet{
				"pod0": sets.String{
					"container0": struct{}{},
				},
			},
			regionInfo: types.RegionInfo{
				RegionName:   "share-xxx",
				RegionType:   types.QoSRegionTypeShare,
				BindingNumas: machine.NewCPUSet(0),
			},
			resourceEssentials: types.ResourceEssentials{
				EnableReclaim:       true,
				ResourceUpperBound:  90,
				ResourceLowerBound:  4,
				ReservedForAllocate: 0,
			},
			controlEssentials: types.ControlEssentials{
				ControlKnobs: types.ControlKnob{
					types.ControlKnobNonReclaimedCPUSize: {
						Value:  40,
						Action: types.ControlKnobActionNone,
					},
				},
				Indicators: types.Indicator{
					consts.MetricCPUSchedwait: {
						Current: 4,
						Target:  400,
					},
				},
				ReclaimOverlap: false,
			},
			wantResult: types.ControlKnob{
				types.ControlKnobNonReclaimedCPUSize: {
					Value:  38,
					Action: types.ControlKnobActionNone,
				},
			},
		},
		{
			name: "share_deadband",
			podSet: types.PodSet{
				"pod0": sets.String{
					"container0": struct{}{},
				},
			},
			regionInfo: types.RegionInfo{
				RegionName:   "share-xxx",
				RegionType:   types.QoSRegionTypeShare,
				BindingNumas: machine.NewCPUSet(0),
			},
			resourceEssentials: types.ResourceEssentials{
				EnableReclaim:       true,
				ResourceUpperBound:  90,
				ResourceLowerBound:  4,
				ReservedForAllocate: 0,
			},
			controlEssentials: types.ControlEssentials{
				ControlKnobs: types.ControlKnob{
					types.ControlKnobNonReclaimedCPUSize: {
						Value:  40,
						Action: types.ControlKnobActionNone,
					},
				},
				Indicators: types.Indicator{
					consts.MetricCPUSchedwait: {
						Current: 401,
						Target:  400,
					},
				},
				ReclaimOverlap: false,
			},
			wantResult: types.ControlKnob{
				types.ControlKnobNonReclaimedCPUSize: {
					Value:  40,
					Action: types.ControlKnobActionNone,
				},
			},
		},
		{
			name: "dedicated_numa_exclusive",
			podSet: types.PodSet{
				"pod0": sets.String{
					"container0": struct{}{},
				},
			},
			regionInfo: types.RegionInfo{
				RegionName:   "dedicated-numa-exclusive-xxx",
				RegionType:   types.QoSRegionTypeDedicatedNumaExclusive,
				BindingNumas: machine.NewCPUSet(0),
			},
			resourceEssentials: types.ResourceEssentials{
				EnableReclaim:       true,
				ResourceUpperBound:  90,
				ResourceLowerBound:  4,
				ReservedForAllocate: 0,
			},
			controlEssentials: types.ControlEssentials{
				ControlKnobs: types.ControlKnob{
					types.ControlKnobNonReclaimedCPUSize: {
						Value:  40,
						Action: types.ControlKnobActionNone,
					},
				},
				Indicators: types.Indicator{
					consts.MetricCPUCPIContainer: {
						Current: 2.0,
						Target:  1.0,
					},
					consts.MetricMemBandwidthNuma: {
						Current: 4,
						Target:  40,
					},
				},
				ReclaimOverlap: false,
			},
			wantResult: types.ControlKnob{
				types.ControlKnobNonReclaimedCPUSize: {
					Value:  48,
					Action: types.ControlKnobActionNone,
				},
			},
		},
	}

	checkpointDir, err := os.MkdirTemp("", "checkpoint")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(checkpointDir) }()

	stateFileDir, err := os.MkdirTemp("", "statefile")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(stateFileDir) }()

	checkpointManagerDir, err := os.MkdirTemp("", "checkpointmanager")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(checkpointManagerDir) }()

	for _, tt := range tests {
		policy := newTestPolicyRama(t, checkpointDir, stateFileDir, checkpointManagerDir, tt.regionInfo, tt.podSet).(*PolicyRama)
		assert.NotNil(t, policy)

		podNames := []string{}
		for podName, containerSet := range tt.podSet {
			podNames = append(podNames, podName)
			for containerName := range containerSet {
				err = metaCacheRama.AddContainer(podName, containerName, &types.ContainerInfo{})
				assert.Nil(t, err)
			}
		}
		policy.metaServer.MetaAgent.SetPodFetcher(constructPodFetcherRama(podNames))

		t.Run(tt.name, func(t *testing.T) {
			policy.SetEssentials(tt.resourceEssentials, tt.controlEssentials)
			policy.Update()
			controlKnobUpdated, err := policy.GetControlKnobAdjusted()

			assert.NoError(t, err)
			assert.Equal(t, tt.wantResult, controlKnobUpdated)
		})
	}
}
