package main

import "log"
import "testing"

func TestEstimateAvailableMemory(t *testing.T) {
	r := FileReaderStub{}
	o := MeminfoObserver{nil, r, 4096, false, false}
	memInfoData, err := o.reader.getFloatKeyValuePairs(meminfoFile)
	if err != nil {
		t.Fatal(err)
	}

	lowWatermarkPages, err := o.getLowPages()
	if err != nil {
		t.Fatal(err)
	}

	/*
		See https://git.kernel.org/pub/scm/linux/kernel/git/torvalds/linux.git/commit/?id=34e431b0ae398fc54ea69ff85ec700722c9da773
		for MemAvailable calculation details
		New method uses 'totalreserve_pages' instead of wmark_low in first 'available' calculation step,
		see  https://github.com/torvalds/linux/blob/master/mm/page_alloc.c 'calculate_totalreserve_pages()'

		wmark_low += zone->watermark[WMARK_LOW];
		wmark_low = (11 + 2656 + 67067 + 0 + 0) * 4096 / 1024 = 69734 * 4096 / 1024 = 278936

		available = i.freeram - wmark_low;
		available = 13506444  - 278936 = 13227508

		pagecache = pages[LRU_ACTIVE_FILE] + pages[LRU_INACTIVE_FILE];
		pagecache -= min(pagecache / 2, wmark_low);
		available += pagecache;

		pagecache = 6418124 + 5018440 = 11436564 +
		pagecache -= min (5718282, 278936) -= 278936
		pagecache = 11157628
		available = 13227508 + 11157628 = 24385136

		available += global_page_state(NR_SLAB_RECLAIMABLE) -
					 min(global_page_state(NR_SLAB_RECLAIMABLE) / 2, wmark_low);

		available += 2069212 - min(2069212, 278936)
		available += 1790276
		available = 24385136 + 1790276 = 26175412

	*/
	memAvailableEstimatedKb := o.estimateAvailableMemory(memInfoData, lowWatermarkPages)

	const expectedEstimatedMemKb = 26175412
	if int64(memAvailableEstimatedKb) != expectedEstimatedMemKb {
		t.Errorf("MemAvailable estimation is incorrect, expected %d, got %d", expectedEstimatedMemKb, int64(memAvailableEstimatedKb))
	}

	procAvailMemValue, ok := memInfoData["MemAvailable"]
	if ok {
		diff := int64(memAvailableEstimatedKb) - int64(procAvailMemValue)
		log.Printf("Diff between old-style and new-style MemAvailable: %d kb - %d kb = %d kb", int64(memAvailableEstimatedKb), int64(procAvailMemValue), diff)
	}
}
