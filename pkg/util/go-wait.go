package util

import "sync"

var wg sync.WaitGroup

func GoWait(goRoutine func(...interface{}) interface{}) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		goRoutine()
	}()
}

func WaitAllGoroutines() {
	wg.Wait()
}
