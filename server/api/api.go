package api

import "sync"

func Serve() {
	var wg sync.WaitGroup

	ServeSocketAPI(&wg)
	ServeRestAPI(&wg)

	wg.Wait()
}
