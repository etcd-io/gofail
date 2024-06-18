package main

import (
	"log"
	"sync"
	"time"

	"go.etcd.io/gofail/integration/sleep/failpoints"
	gofail "go.etcd.io/gofail/runtime"
)

func main() {
	{
		// expectation: this part of the code will take about 3s to execute, because all go routines will be executing concurrently
		start := time.Now()

		log.Println("Stage 1: Run 3 workers under normal logic")
		var wg sync.WaitGroup
		wg.Add(1)
		go failpoints.Worker1(&wg)

		wg.Add(1)
		go failpoints.Worker2(&wg)

		wg.Add(1)
		go failpoints.Worker3(&wg)

		wg.Wait()

		elapsed := time.Since(start)
		if elapsed > (3*time.Second + 100*time.Millisecond) {
			log.Fatalln("invalid execution time", elapsed)
		}

		log.Println("Stage 1: Done")
	}

	{
		// expectation: this part of the code will take about 6s to execute only,
		// because all go routines will be executing concurrently, with both the sleep
		// from failpoint and the original sleep actions
		//
		// The gofail implementation up till commit 93c579a86c46 is executing the
		// program sequentially, due to the failpoint action execution and enable/disable
		// flows are under the same locking mechanism, only one of the actions can make
		// progress at a given moment
		log.Println("Stage 2: Run 3 workers under failpoint logic")

		start := time.Now()

		var wg sync.WaitGroup
		gofail.Enable("worker1Failpoint", `sleep("3s")`)
		wg.Add(1)
		go failpoints.Worker1(&wg)
		time.Sleep(10 * time.Millisecond)

		gofail.Enable("worker2Failpoint", `sleep("3s")`)
		wg.Add(1)
		go failpoints.Worker2(&wg)
		time.Sleep(10 * time.Millisecond)

		gofail.Enable("worker3Failpoint", `sleep("3s")`)
		wg.Add(1)
		go failpoints.Worker3(&wg)
		time.Sleep(10 * time.Millisecond)

		// the failpoint can be disabled during failpoint execution
		gofail.Disable("worker1Failpoint")
		gofail.Disable("worker2Failpoint")
		gofail.Disable("worker3Failpoint")

		wg.Wait()

		elapsed := time.Since(start)
		if elapsed > (6*time.Second + 100*time.Millisecond) {
			log.Fatalln("invalid execution time", elapsed)
		}

		log.Println("Stage 2: Done")
	}
}
