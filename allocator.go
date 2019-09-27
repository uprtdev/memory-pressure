package main

import (
	"log"
	"math/rand"
	"time"
)

type Allocator struct {
	tracker *Tracker
	blob    []*[]byte
	total   int
}

func (f *Allocator) allocateBlock(blockSizeInMb int) {
	const bytesInMb int = 1024 * 1024
	totalSize := blockSizeInMb * bytesInMb
	pageSize := PageSize()
	newArr := make([]byte, totalSize)

	step := totalSize / pageSize
	for i := 0; i < step; i++ {
		newArr[i*pageSize] = byte(rand.Intn(255))
	}
	f.blob = append(f.blob, &newArr)
}

func (f *Allocator) initialize(t *Tracker, initialBlockSizeMb int) {
	f.tracker = t
	f.tracker.trackOne("alloctd", 0)
	if initialBlockSizeMb > 0 {
		log.Printf("Pre-allocating initial block")
		f.allocateBlock(initialBlockSizeMb)
		log.Printf("Allocated, size is %v Mb", initialBlockSizeMb)
		f.total = initialBlockSizeMb
		f.tracker.trackOne("alloctd", initialBlockSizeMb)
	}
}

func (f *Allocator) startMemoryFilling(blockSizeInMb int, period time.Duration, limit int) {
	ticker := time.NewTicker(period)
	for f.total < limit {
		f.allocateBlock(blockSizeInMb)
		f.total = f.total + blockSizeInMb
		f.tracker.trackOne("alloctd", f.total)
		<-ticker.C
	}
	log.Printf("Allocated %v Mb, maximum limit is set to %v, stopping allocation process...", f.total, limit)
}
