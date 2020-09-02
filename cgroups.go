package main

/*
#include <sys/eventfd.h>
*/
import "C"

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"syscall"
)

const eventControlPath = "/sys/fs/cgroup/memory/cgroup.event_control"
const pressureLevelPath = "/sys/fs/cgroup/memory/memory.pressure_level"

type CgroupsObserver struct {
	tracker         *Tracker
	notifyChan      chan bool
	lowEventFd      int
	mediumEventFd   int
	criticalEventFd int
	lowLevel        bool
	mediumLevel     bool
	criticalLevel   bool
	oldPressure     int
}

func (o *CgroupsObserver) createFd() (int, error) {
	fd, err := C.eventfd(0, C.EFD_CLOEXEC)
	if err != nil {
		log.Fatal(err)
	}
	return int(fd), err
}

func (o *CgroupsObserver) enableNotification(eventfd int, level string) error {

	fileEventControl, err := os.OpenFile(eventControlPath, os.O_RDWR, 0755)
	if err != nil {
		return err
	}
	defer fileEventControl.Close()

	filePressure, err := os.Open(pressureLevelPath)
	if err != nil {
		return err
	}
	defer filePressure.Close()

	command := fmt.Sprintf("%d %d %s", eventfd, filePressure.Fd(), level)
	_, err = fileEventControl.WriteString(command)
	return err
}

func (o *CgroupsObserver) SetFlags() {

}

func (o *CgroupsObserver) Initialize(t *Tracker, r Reader, c chan bool) {
	o.tracker = t
	o.notifyChan = c
	o.oldPressure = 0xFF
	var err error
	atLeastOne := false

	o.lowEventFd, err = o.createFd()
	if err == nil {
		err = o.enableNotification(o.lowEventFd, "low")
	}
	if err != nil {
		log.Print(err)
	} else {
		atLeastOne = true
		go o.startCheckingPressure(o.lowEventFd, &o.lowLevel)
	}

	o.mediumEventFd, err = o.createFd()
	if err == nil {
		err = o.enableNotification(o.mediumEventFd, "medium")
	}
	if err != nil {
		log.Print(err)
	} else {
		atLeastOne = true
		go o.startCheckingPressure(o.mediumEventFd, &o.mediumLevel)
	}

	o.criticalEventFd, err = o.createFd()
	if err == nil {
		err = o.enableNotification(o.criticalEventFd, "critical")
	}
	if err != nil {
		log.Print(err)
	} else {
		atLeastOne = true
		go o.startCheckingPressure(o.criticalEventFd, &o.criticalLevel)
	}

	if atLeastOne == true {
		o.reportPressureIfChanged()
	}
}

func (o *CgroupsObserver) Close() error {
	syscall.Close(o.lowEventFd)
	syscall.Close(o.mediumEventFd)
	syscall.Close(o.criticalEventFd)
	return nil
}

func (o *CgroupsObserver) startCheckingPressure(eventfd int, target *bool) {
	buf := make([]byte, 8)
	for {
		n, err := syscall.Read(eventfd, buf[:])
		if err != nil {
			log.Print(err)
		}
		val, n := binary.Uvarint(buf)
		if n > 0 {
			if val > 0 {
				*target = true
			} else {
				*target = false
			}
		}
		o.reportPressureIfChanged()
	}
}

func (o *CgroupsObserver) reportPressureIfChanged() {
	const pressureKey = "cgroups"
	newPressure := 0
	if o.criticalLevel {
		newPressure |= 1 << 2
	}
	if o.mediumLevel {
		newPressure |= 1 << 1
	}
	if o.lowLevel {
		newPressure |= 1 << 0
	}

	if newPressure != o.oldPressure {
		o.oldPressure = newPressure
		o.tracker.trackOne(pressureKey, newPressure)

		// non-nlocking notification sending
		select {
		case o.notifyChan <- true:
		default:
		}
	}
}
