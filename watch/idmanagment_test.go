package watch

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateID(t *testing.T) {
	s0 := CreateID()
	fmt.Printf("%d\n", s0)
	s1 := CreateID()
	fmt.Printf("%d\n", s1)

	assert.NotEqual(t, s1, s0, "ids equal")

	s2 := CreateID()
	fmt.Printf("%d\n", s2)
	assert.NotEqual(t, s1, s2, "ids equal")
	assert.NotEqual(t, s2, s0, "ids equal")
}
