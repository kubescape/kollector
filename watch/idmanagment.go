package watch

import (
	"container/list"
	"sync"
)

type IDDataBase struct {
	Ids     *list.List
	Mutex   sync.Mutex
	counter int
}

var ids IDDataBase = IDDataBase{Ids: list.New(), counter: 0}

//static function
func IDCreator() int {
	id := ids.counter
	ids.counter++
	return id
}

func CreateID() int {

	var id int
	var flag int = 1

	for flag == 1 {
		flag = 0
		id = IDCreator()
		for e := ids.Ids.Front(); e != nil; e = e.Next() {
			if e.Value.(int) == id {
				flag = 1
				break
			}
		}

	}
	ids.Mutex.Lock()
	ids.Ids.PushBack(id)
	ids.Mutex.Unlock()
	return id
}

func DeleteID(id int) {
	for e := ids.Ids.Front(); e != nil; e = e.Next() {
		if e.Value.(int) == id {
			ids.Ids.Remove(e)
			break
		}
	}
}
