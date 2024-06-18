package failpoints

import (
	"log"
	"sync"
	"time"
)

func Worker1(wg *sync.WaitGroup) {
	defer wg.Done()

	log.Println("worker1 in")
	defer log.Println("worker1 out")

	// gofail: var worker1Failpoint struct{}

	time.Sleep(3 * time.Second)
}

func Worker2(wg *sync.WaitGroup) {
	defer wg.Done()

	log.Println("worker2 in")
	defer log.Println("worker2 out")

	// gofail: var worker2Failpoint struct{}

	time.Sleep(3 * time.Second)
}

func Worker3(wg *sync.WaitGroup) {
	defer wg.Done()

	log.Println("worker3 in")
	defer log.Println("worker3 out")

	// gofail: var worker3Failpoint struct{}

	time.Sleep(3 * time.Second)
}
