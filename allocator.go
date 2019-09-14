package main

import (
	"math/rand"
	"time"
)

type Allocator struct {
	tracker *Tracker
	blob    []*[]byte
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

func (f *Allocator) initialize(t *Tracker) {
	f.tracker = t
	f.tracker.trackOne("alloctd", 0)
}

func (f *Allocator) startMemoryFilling(blockSizeInMb int) {
	ticker := time.NewTicker(1 * time.Second)
	for {
		f.allocateBlock(blockSizeInMb)
		f.tracker.trackOne("alloctd", len(f.blob)*blockSizeInMb)
		<-ticker.C
	}
}
