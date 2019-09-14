package main

import "log"
import "strconv"
import "os"
import "testing"

func TestFillOnce(t *testing.T) {
	const step1BlockSizeInMb = 512
	const step2BlockSizeInMb = 1024

	if testing.Short() {
		log.Print("Skipping memory block allocator test in short mode")
		t.Skip()
	}

	f := Allocator{}
	r := FileReader{}

	procStatusFile := "/proc/" + strconv.FormatInt(int64(os.Getpid()), 10) + "/status"

	vmSizeBegin, err := r.getIntValue(procStatusFile, "VmSize")
	if err != nil {
		t.Fatal(err)
	}

	f.allocateBlock(step1BlockSizeInMb)
	vmSizeMiddle, err := r.getIntValue(procStatusFile, "VmSize")
	if err != nil {
		t.Fatal(err)
	}
	memoryGrowth := int64(vmSizeMiddle - vmSizeBegin)
	if (memoryGrowth / 1024) < step1BlockSizeInMb {
		t.Errorf("Don't see any memory growth on step 1, allocated %d Mb, growth is %d Mb", step1BlockSizeInMb, memoryGrowth/1024)
	}

	f.allocateBlock(step2BlockSizeInMb)
	vmSizeFinish, err := r.getIntValue(procStatusFile, "VmSize")
	if err != nil {
		t.Fatal(err)
	}

	memoryGrowth = int64(vmSizeFinish - vmSizeMiddle)
	if (memoryGrowth / 1024) < step2BlockSizeInMb {
		t.Errorf("Don't see any memory growth on step 2, allocated %d Mb, growth is %d Mb", step2BlockSizeInMb, memoryGrowth/1024)
	}
}
