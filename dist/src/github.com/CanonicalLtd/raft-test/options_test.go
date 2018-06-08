// Copyright 2017 Canonical Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rafttest_test

/*
// The Latency options tweak the default raft timeouts.
func TestCluster_Latency(t *testing.T) {
	rafts, control := rafttest.Cluster(t, rafttest.FSMs(3), rafttest.Latency(1000.0))
	defer control.Close()

	time.Sleep(250 * time.Millisecond)

	// Since we have a very high latency the election shouldn't have been
	// performed yet, despite the above sleep.
	for i := range rafts {
		assert.NotEqual(t, raft.Leader, rafts[i].State())
	}
}

// If a GO_RAFT_TEST_LATENCY environment variable is set, the default raft
// timeouts are changed accordingly.
func TestCluster_LatencyEnv(t *testing.T) {
	if env, ok := os.LookupEnv("GO_RAFT_TEST_LATENCY"); ok {
		defer os.Setenv("GO_RAFT_TEST_LATENCY", env)
	} else {
		defer os.Unsetenv("GO_RAFT_TEST_LATENCY")
	}
	os.Setenv("GO_RAFT_TEST_LATENCY", "1000.0")

	rafts, control := rafttest.Cluster(t, rafttest.FSMs(3))
	defer control.Close()

	time.Sleep(250 * time.Millisecond)

	// Since we have a very high latency the election shouldn't have been
	// performed yet, despite the above sleep.
	for i := range rafts {
		assert.NotEqual(t, raft.Leader, rafts[i].State())
	}
}

// If the GO_RAFT_TEST_LATENCY environment variable is not set with a float
// number, the test fails.
func TestCluster_LatencyEnvInvalid(t *testing.T) {
	if env, ok := os.LookupEnv("GO_RAFT_TEST_LATENCY"); ok {
		defer os.Setenv("GO_RAFT_TEST_LATENCY", env)
	} else {
		defer os.Unsetenv("GO_RAFT_TEST_LATENCY")
	}
	os.Setenv("GO_RAFT_TEST_LATENCY", "foo")

	f := func() { rafttest.Cluster(&testing.T{}, rafttest.FSMs(3)) }

	assert.PanicsWithValue(t, "invalid value 'foo' for GO_RAFT_TEST_LATENCY", f)
}
*/

/*
// If the Servers option is used, only the given nodes are connected and
// bootstrapped.
func TestServers(t *testing.T) {
	fsms := rafttest.FSMs(3)
	servers := rafttest.Servers(0)
	rafts, control := rafttest.Cluster(t, fsms, servers)
	defer control.Close()

	assert.Equal(t, rafts[0], control.LeadershipAcquired(time.Second))

	assert.Equal(t, raft.Leader, rafts[0].State())
	future := rafts[0].GetConfiguration()
	require.NoError(t, future.Error())
	assert.Len(t, future.Configuration().Servers, 1)
}
*/
