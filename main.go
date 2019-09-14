package main

/*
#include <unistd.h>
*/
import "C"

import (
	"os"
	"os/signal"
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
