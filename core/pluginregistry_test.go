// Copyright 2015 trivago GmbH
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

package core

import (
	"github.com/trivago/gollum/shared"
	"testing"
)

func TestPluginRegistry(t *testing.T) {
	expect := shared.NewExpect(t)
	plugin, err := NewPlugin(NewPluginConfig("randomPlugin"))
	expect.NotNil(err)

	// Test for Register
	PluginRegistry.Register(plugin, "aPlugin")
	expect.Equal(1, len(PluginRegistry.plugins))

	// Test for RegisterUnique
	PluginRegistry.RegisterUnique(plugin, "aPlugin")
	expect.Equal(1, len(PluginRegistry.plugins))

	// Test for GetPlugin
	ret := PluginRegistry.GetPlugin("nonExistentPlugin")
	expect.Nil(ret)
	ret = PluginRegistry.GetPlugin("aPlugin")
	expect.Equal(plugin, ret)

	// Test for GetPluginWithState
	ret = PluginRegistry.GetPluginWithState("aPlugin")
	expect.Nil(ret)
	// TODO: create mock PluginState with state and then test notnil
}
