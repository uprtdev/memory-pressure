package main

/*
#include <unistd.h>
*/
import "C"

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func PageSize() int {
	return int(C.sysconf(C._SC_PAGE_SIZE))
}

func ParseCustomParams(args string) map[string]string {
	result := make(map[string]string)
	pairs := strings.Split(args, ",")
	for _, option := range pairs {
		pair := strings.Split(option, "=")
		if len(pair) == 2 {
			result[pair[0]] = pair[1]
		}
	}
	return result
}

type PassiveObserver interface {
	Initialize(t *Tracker, r Reader, p map[string]string)
	TimerEvent()
}

type ActiveObserver interface {
	Initialize(t *Tracker, r Reader, c chan bool, p map[string]string)
}

func main() {
	var blockSizeInMb = flag.Int("blockSize", 128, "block size for every allocation (in Mb), 0 to disable periodical allocator")
	var initialBlockSizeInMb = flag.Int("initialSize", 0, "size to allocate before test start (in Mb), 0 to disable initial allocation")
	var allocatePeriodInS = flag.Int("allocInterval", 1, "time delay between allocations (in seconds)")
	var printPeriodInS = flag.Int("printInterval", 5, "time delay between current status updates (in seconds)")
	var maximumLimitInMb = flag.Int("limit", 0, "maximum allocated memory size (in Mb), 0 to disable the limit")
	var customParams = flag.String("options", "", "custom options for observers, see README for information")
	flag.Parse()
	observersOptions := ParseCustomParams(*customParams)

	r := FileReader{}
	t := Tracker{}
	t.prepare()

	var passiveObservers []PassiveObserver
	passiveObservers = append(passiveObservers, &MeminfoObserver{})
	passiveObservers = append(passiveObservers, &SwapObserver{})
	passiveObservers = append(passiveObservers, &PsiObserver{})
	for _, element := range passiveObservers {
		element.Initialize(&t, r, observersOptions)
	}

	notifySink := make(chan bool)
	var activeObservers []ActiveObserver
	activeObservers = append(activeObservers, &CgroupsObserver{})
	for _, element := range activeObservers {
		element.Initialize(&t, r, notifySink, observersOptions)
	}

	if *blockSizeInMb == 0 {
		log.Printf("Working in a passive mode, will not allocate memory during the test")
	}

	if *blockSizeInMb > 0 || *initialBlockSizeInMb > 0 {
		a := Allocator{}
		a.initialize(&t, *initialBlockSizeInMb)
		if *blockSizeInMb > 0 {
			log.Printf("Will allocate %v Mb every %v seconds", *blockSizeInMb, *allocatePeriodInS)
			go a.startMemoryFilling(*blockSizeInMb, time.Duration(*allocatePeriodInS)*time.Second, *maximumLimitInMb)
		}
	}

	t.prepareAndPrintHeader()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(time.Duration(*printPeriodInS) * time.Second)

	for {
		select {
		case <-sig:
			os.Exit(0)
		case <-notifySink:
		case <-ticker.C:
		}
		for _, element := range passiveObservers {
			element.TimerEvent()
		}
		t.saveTime()
		t.printData()
	}

}
