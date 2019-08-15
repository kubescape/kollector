package watch

import (
	"fmt"
	"testing"
)

func TestIdsManagment1(test *testing.T) {
	s0 := CreateID()
	fmt.Printf("%d\n", s0)
	s1 := CreateID()
	fmt.Printf("%d\n", s1)
	if s1 == s0 {
		test.Errorf("ids equal s0 s1")
	}

	s2 := CreateID()
	fmt.Printf("%d\n", s2)
	if s1 == s2 {
		test.Errorf("ids equal s2 s1")
	}
	if s2 == s0 {
		test.Errorf("ids equal s0 s2")
	}

}
