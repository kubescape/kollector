package watch

import (
	"fmt"
	"testing"
)

func TestIdsManagment1(test *testing.T) {
	s := CreateID()
	fmt.Printf("%d\n", s)
	s = CreateID()
	fmt.Printf("%d\n", s)
	s = CreateID()
	fmt.Printf("%d\n", s)
	s = CreateID()
	fmt.Printf("%d\n", s)
	s = CreateID()
	fmt.Printf("%d\n", s)
	s = CreateID()
	fmt.Printf("%d\n", s)
	test.Errorf("123")
}
