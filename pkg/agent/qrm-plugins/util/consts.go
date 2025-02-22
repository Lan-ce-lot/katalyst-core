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

package util

const (
	// common metrics for all types of qrm plugins
	MetricNameHeartBeat               = "heartbeat"
	MetricNameAllocateFailed          = "alloc_failed"
	MetricNameGetTopologyHintsFailed  = "get_topology_hints_failed"
	MetricNameRemovePodFailed         = "remove_pod_failed"
	MetricNameLWAdvisorServerFailed   = "lw_advisor_server_failed"
	MetricNameHandleAdvisorRespCalled = "handle_advisor_resp_called"
	MetricNameHandleAdvisorRespFailed = "handle_advisor_resp_failed"
	MetricNameLWRecvStuck             = "lw_recv_stuck"

	// metrics for cpu plugin
	MetricNamePoolSize         = "pool_size"
	MetricNameRealStateInvalid = "real_state_invalid"
	MetricNameCPUSetInvalid    = "cpuset_invalid"
	MetricNameCPUSetOverlap    = "cpuset_overlap"

	// metrics for memory plugin
	MetricNameMemSetInvalid                           = "memset_invalid"
	MetricNameMemSetOverlap                           = "memset_overlap"
	MetricNameNodeMemsetInvalid                       = "node_memset_invalid"
	MetricNameMemoryHandleAdvisorContainerEntryFailed = "memory_handle_advisor_container_entry_failed"
	MetricNameMemoryHandleAdvisorExtraEntryFailed     = "memory_handle_advisor_extra_entry_failed"
	MetricNameMemoryHandleAdvisorMemoryLimit          = "memory_handle_advisor_memory_limit"
	MetricNameMemoryHandleAdvisorDropCache            = "memory_handle_advisor_drop_cache"
	MetricNameMemoryHandleAdvisorCPUSetMems           = "memory_handle_advisor_cpuset_mems"
	MetricNameMemoryOOMPriorityDeleteFailed           = "memory_oom_priority_delete_failed"
	MetricNameMemoryOOMPriorityUpdateFailed           = "memory_oom_priority_update_failed"
)

// those are OCI property names to be used by QRM plugins
const (
	OCIPropertyNameCPUSetCPUs         = "CpusetCpus"
	OCIPropertyNameCPUSetMems         = "CpusetMems"
	OCIPropertyNameMemoryLimitInBytes = "MemoryLimitInBytes"
)

const QRMTimeFormat = "2006-01-02 15:04:05.999999999 -0700 MST"

const QRMPluginPolicyTagName = "policy"
