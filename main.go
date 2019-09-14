package main

/*
#include <unistd.h>
*/
import "C"

import (
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

func PageSize() int {
	return int(C.sysconf(C._SC_PAGE_SIZE))
}

type PassiveObserver interface {
	Initialize(t *Tracker, r Reader)
	TimerEvent()
}

type ActiveObserver interface {
	Initialize(t *Tracker, r Reader, c chan bool)
}

func main() {
	var blockSizeInMb = 128
	params := os.Args[1:]
	if len(params) > 0 {
		userBlockSize, err := strconv.ParseInt(params[0], 10, 64)
		if err == nil {
			blockSizeInMb = int(userBlockSize)
		} else {
			log.Printf("Failed to parse block size: '%v'", params[0])
		}
	}

	r := FileReader{}
	t := Tracker{}
	t.prepare()

	var passiveObservers []PassiveObserver
	passiveObservers = append(passiveObservers, &MeminfoObserver{})
	passiveObservers = append(passiveObservers, &SwapObserver{})
	passiveObservers = append(passiveObservers, &PsiObserver{})
	for _, element := range passiveObservers {
		element.Initialize(&t, r)
	}

	notifySink := make(chan bool)
	var activeObservers []ActiveObserver
	activeObservers = append(activeObservers, &CgroupsObserver{})
	for _, element := range activeObservers {
		element.Initialize(&t, r, notifySink)
	}

	if blockSizeInMb > 0 {
		log.Printf("Using block size %v Mb", blockSizeInMb)
		a := Allocator{}
		a.initialize(&t)
		go a.startMemoryFilling(blockSizeInMb)
	} else {
		log.Printf("Working in a passive mode, will not allocate memory during the test")
	}

	t.prepareAndPrintHeader()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(5 * time.Second)

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
