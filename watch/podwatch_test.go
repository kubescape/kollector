package watch

import "testing"

func TestPodWatch(test *testing.T) {
	wh := CreateWatchHandler()

	go func() {
		wh.PodWatch()
	}()


	
}
