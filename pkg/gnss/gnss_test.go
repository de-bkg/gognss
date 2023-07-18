// Package gnss contains common constants and type definitions.
package gnss

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSystems_MarshalJSON(t *testing.T) {
	systems := Systems{SysGAL, SysBDS}
	sysJSON, err := json.Marshal(systems)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "[\"E\",\"C\"]", string(sysJSON), "marshall gnss")
}
