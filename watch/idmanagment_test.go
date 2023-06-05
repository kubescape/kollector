package watch

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateID(t *testing.T) {
	s0 := CreateID()
	s1 := CreateID()

	assert.NotEqual(t, s1, s0, "ids equal")

	s2 := CreateID()

	assert.NotEqual(t, s1, s2, "ids equal")
	assert.NotEqual(t, s2, s0, "ids equal")
}
