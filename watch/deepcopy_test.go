package watch

import (
	"reflect"
	"testing"
)

type Person struct {
	Name    string
	Age     int
	Address Address
}

type Address struct {
	Street  string
	City    string
	Country string
}

func TestClone(t *testing.T) {
	src := &Person{
		Name: "Alice",
		Age:  30,
		Address: Address{
			Street:  "123 Main St",
			City:    "Anytown",
			Country: "USA",
		},
	}
	var dst Person
	err := Clone(src, &dst)
	if err != nil {
		t.Errorf("Clone error: %v", err)
	}
	if !reflect.DeepEqual(src, &dst) {
		t.Errorf("Clone result not equal to source")
	}
}

func TestDeepCopy(t *testing.T) {
	src := &Person{
		Name: "Bob",
		Age:  40,
		Address: Address{
			Street:  "456 Elm St",
			City:    "Othertown",
			Country: "USA",
		},
	}
	var dst Person
	err := DeepCopy(src, &dst)
	if err != nil {
		t.Errorf("DeepCopy error: %v", err)
	}
	if !reflect.DeepEqual(src, &dst) {
		t.Errorf("DeepCopy result not equal to source")
	}
}

func TestDeepCopyObj(t *testing.T) {
	src := &Person{
		Name: "Charlie",
		Age:  50,
		Address: Address{
			Street:  "789 Oak St",
			City:    "Somewhere",
			Country: "USA",
		},
	}
	dst := new(Person)
	err := DeepCopyObj(src, dst)
	if err != nil {
		t.Errorf("DeepCopyObj error: %v", err)
	}
	if !reflect.DeepEqual(src, dst) {
		t.Errorf("DeepCopyObj result not equal to source")
	}
}
