package watch

import (
	"testing"
)

func TestIdsManagment1(test *testing.T) {
<<<<<<< HEAD
	s0 := CreateID()
	fmt.Printf("%d\n", s0)
	s1 := CreateID()
	fmt.Printf("%d\n", s1)
	if s0 == s1 {
		test.Errorf("s0 s1 ids equal")
	}
	s2 := CreateID()
	fmt.Printf("%d\n", s2)
	if s0 == s2 {
		test.Errorf("s0 s2 ids equal")
	}
	if s2 == s1 {
		test.Errorf("s2 s1 ids equal")
	}
=======
	// s := CreateID()
	// fmt.Printf("%d\n", s)
	// s = CreateID()
	// fmt.Printf("%d\n", s)
	// s = CreateID()
	// fmt.Printf("%d\n", s)
	// s = CreateID()
	// fmt.Printf("%d\n", s)
	// s = CreateID()
	// fmt.Printf("%d\n", s)
	// s = CreateID()
	// fmt.Printf("%d\n", s)
	// test.Errorf("123")
>>>>>>> 4585080cdfe5a3fd4568c5d02ff0ca7782f593a6
}
