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

package logging

import (
	"log"
	"testing"

	"github.com/hashicorp/logutils"
)

// New returns a standard log.Logger that will write entries at or above the
// specified level to the testing log.
func New(t testing.TB, level logutils.LogLevel) *log.Logger {
	filter := &logutils.LevelFilter{
		Levels:   []logutils.LogLevel{"DEBUG", "WARN", "ERROR", "INFO"},
		MinLevel: level,
		Writer:   &testingWriter{t},
	}
	return log.New(filter, "", log.Ltime|log.Lmicroseconds)
}

// Implement io.Writer and forward what it receives to a
// testing logger.
type testingWriter struct {
	t testing.TB
}

// Write a single log entry. It's assumed that p is always a \n-terminated UTF
// string.
func (w *testingWriter) Write(p []byte) (n int, err error) {
	w.t.Logf(string(p))
	return len(p), nil
}
