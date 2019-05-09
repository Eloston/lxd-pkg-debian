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

package trace_test

import (
	"testing"

	"github.com/CanonicalLtd/go-dqlite/internal/trace"
	"github.com/stretchr/testify/assert"
)

func TestString_String(t *testing.T) {
	s := trace.String("foo", "bar")
	assert.Equal(t, "foo=bar", s.String())
}

func TestInteger_String(t *testing.T) {
	i := trace.Integer("foo", 123)
	assert.Equal(t, "foo=123", i.String())
}