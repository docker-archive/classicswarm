#!/bin/bash

# Copyright 2015 The Kubernetes Authors All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

## Contains configuration values for the CentOS cluster
# The user should have sudo privilege
export MASTER=${MASTER:-"centos@172.10.0.11"}
export MASTER_IP=${MASTER#*@}

# Define all your minion nodes,
# And separated with blank space like <user_1@ip_1> <user_2@ip_2> <user_3@ip_3>.
# The user should have sudo privilege
export MINIONS=${MINIONS:-"centos@172.10.0.12 centos@172.10.0.13"}
# If it practically impossible to set an array as an environment variable
# from a script, so assume variable is a string then convert it to an array
export MINIONS_ARRAY=($MINIONS)

# Number of nodes in your cluster.
export NUM_MINIONS=${NUM_MINIONS:-2}

# By default, the cluster will use the etcd installed on master.
export ETCD_SERVERS=${ETCD_SERVERS:-"http://$MASTER_IP:4001"}

# define the IP range used for service cluster IPs.
# according to rfc 1918 ref: https://tools.ietf.org/html/rfc1918 choose a private ip range here.
export SERVICE_CLUSTER_IP_RANGE=${SERVICE_CLUSTER_IP_RANGE:-"192.168.3.0/24"}

# define the IP range used for flannel overlay network, should not conflict with above SERVICE_CLUSTER_IP_RANGE
export FLANNEL_NET=${FLANNEL_NET:-"172.16.0.0/16"}

# Admission Controllers to invoke prior to persisting objects in cluster
export ADMISSION_CONTROL=NamespaceLifecycle,NamespaceExists,LimitRanger,ServiceAccount,ResourceQuota,SecurityContextDeny

# Extra options to set on the Docker command line.
# This is useful for setting --insecure-registry for local registries.
export DOCKER_OPTS=${DOCKER_OPTS:-""} 


# Timeouts for process checking on master and minion
export PROCESS_CHECK_TIMEOUT=${PROCESS_CHECK_TIMEOUT:-180} # seconds.
