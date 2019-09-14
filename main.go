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

type Observer interface {
	Initialize(t *Tracker, r Reader)
	TimerEvent()
}

func main() {
	r := FileReader{}
	t := Tracker{}
	t.prepare()

	var observers []Observer
	observers = append(observers, &MeminfoObserver{})
	observers = append(observers, &SwapObserver{})
	observers = append(observers, &PsiObserver{})
	for _, element := range observers {
		element.Initialize(&t, r)
	}

	t.prepareAndPrintHeader()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(5 * time.Second)

	for {
		select {
		case <-sig:
			os.Exit(0)
		case <-ticker.C:
		}
		for _, element := range observers {
			element.TimerEvent()
		}
		t.saveTime()
		t.printData()
	}

}
