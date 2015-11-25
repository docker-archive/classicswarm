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

function start()
{
    /usr/sbin/glusterd -p /run/glusterd.pid
    gluster volume create test_vol `hostname -i`:/vol force
    gluster volume start test_vol
}

function stop()
{
    gluster --mode=script volume stop test_vol force
    kill $(cat /run/glusterd.pid)
    exit 0
}


trap stop TERM

start "$@"

while true; do
 read
done

