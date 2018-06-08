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

package logging_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/CanonicalLtd/raft-test/internal/logging"
)

// Just exercise that nothing breaks, there's no way to inspect
func TestNew(t *testing.T) {
	logger := logging.New(t, "TRACE")
	logger.Printf("[TRACE] raft-test: hello")

	rt := reflect.ValueOf(t).Elem()

	// Assumes that testing.common is the first field of testing.T
	rcommon := rt.Field(0)

	// Asumes that output is the first field of testing.common
	routput := rcommon.Field(1)

	output := string(routput.Bytes())
	if !strings.Contains(output, "[TRACE] raft-test: hello") {
		t.Fatal("logger output not written to testing log")
	}
}
