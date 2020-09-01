package main

import (
	"flag"
	"log"
	"syscall"
)

const (
	psiPath     = "/proc/pressure/memory"
	pressureKey = "psi_trig"
)

type PsiTrigObserver struct {
	tracker             *Tracker
	notifyChan          chan bool
	mediumEventFd       int
	criticalEventFd     int
	oldPressure         int
	timeout             int
	mediumLevelString   string
	criticalLevelString string
}

func (o *PsiTrigObserver) SetFlags() {
	flag.StringVar(&o.mediumLevelString, "psiMediumTrigger", "some 150000 1000000", "PSI medium trigger string")
	flag.StringVar(&o.criticalLevelString, "psiCriticalTrigger", "full 100000 1000000", "PSI critical trigger string")
	flag.IntVar(&o.timeout, "psiTrigTimeout", 5, "PSI trigger timeout")
}

func (o *PsiTrigObserver) Initialize(t *Tracker, r Reader, c chan bool) {
	o.tracker = t
	o.notifyChan = c
	var err error

	o.mediumEventFd, err = o.initializeFd(prepareLevelString(o.mediumLevelString))
	if err != nil {
		log.Print("Error while creating medium PSI trigger: ", err)
		return
	}

	o.criticalEventFd, err = o.initializeFd(prepareLevelString(o.criticalLevelString))
	if err != nil {
		log.Print("Error while creating critical PSI trigger: ", err)
		return
	}

	o.tracker.trackOne(pressureKey, 0)
	go o.startCheckingPressure()
}

func (o *PsiTrigObserver) Close() error {
	syscall.Close(o.mediumEventFd)
	syscall.Close(o.criticalEventFd)
	// closing a file descriptor cause it to be removed from all epoll interest lists
	// so we don't care about EPOLL_CTL_DEL
	return nil
}

func (o *PsiTrigObserver) initializeFd(level []byte) (int, error) {
	fd, err := syscall.Open(psiPath, syscall.O_RDWR|syscall.O_NONBLOCK, 0777)
	if err == nil {
		_, err = syscall.Write(fd, level)
	}
	return fd, err
}

func prepareLevelString(level string) []byte {
	levelString := []byte(level)
	levelString = append(levelString, 0x0) // must be null-terminated
	return levelString
}

func setupPolling(epfd int, fd int) error {
	var event syscall.EpollEvent
	event.Events = syscall.EPOLLPRI
	event.Fd = int32(fd)
	return syscall.EpollCtl(epfd, syscall.EPOLL_CTL_ADD, fd, &event)
}

func (o *PsiTrigObserver) startCheckingPressure() {
	var events [2]syscall.EpollEvent
	epfd, err := syscall.EpollCreate1(0)
	if err != nil {
		log.Print("epoll_create1 failed: ", err)
		return
	}
	defer syscall.Close(epfd)

	if err = setupPolling(epfd, o.mediumEventFd); err != nil {
		log.Print("Failed to setup epoll(): ", err)
	}

	if err = setupPolling(epfd, o.criticalEventFd); err != nil {
		log.Print("Failed to setup epoll(): ", err)
	}

	for {
		var criticalLevel bool
		var mediumLevel bool
		nevents, err := syscall.EpollWait(epfd, events[:], int(o.timeout*1000))
		if err != nil {
			log.Print("epoll_wait failed: ", err)
			break
		}

		for ev := 0; ev < nevents; ev++ {
			if int(events[ev].Fd) == o.criticalEventFd {
				criticalLevel = true
			}
			if int(events[ev].Fd) == o.mediumEventFd {
				mediumLevel = true
			}
		}
		o.reportPressureIfChanged(criticalLevel, mediumLevel)
	}
}

func (o *PsiTrigObserver) reportPressureIfChanged(criticalLevel bool, mediumLevel bool) {
	newPressure := 0
	if criticalLevel {
		newPressure |= 2
	}
	if mediumLevel {
		newPressure |= 1
	}
	if newPressure != o.oldPressure {
		o.tracker.trackOne(pressureKey, newPressure)
		// non-nlocking notification sending
		select {
			case o.notifyChan <- true:
			default:
		}
		o.oldPressure = newPressure
	}
}
