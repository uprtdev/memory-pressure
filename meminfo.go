package main

import (
	"log"
	"math"
)

const zoneFile = "/proc/zoneinfo"
const meminfoFile = "/proc/meminfo"

type MeminfoObserver struct {
	tracker  *Tracker
	reader   Reader
	pageSize int
	params   map[string]string
}

func (o *MeminfoObserver) Initialize(t *Tracker, r Reader, p map[string]string) {
	o.tracker = t
	o.reader = r
	o.params = p
	o.pageSize = PageSize()
	log.Printf("System page size is %d bytes", o.pageSize)
	o.process()
}

func (o *MeminfoObserver) TimerEvent() {
	o.process()
}

func (o *MeminfoObserver) getLowPages() (float64, error) {
	const lowWatermarkKeyword = "low"
	lowWatermarkValues, err := o.reader.getSumAllIntValues(zoneFile, lowWatermarkKeyword)
	if err != nil {
		return 0, err
	}
	var totalLowPages int64
	for _, value := range lowWatermarkValues {
		totalLowPages = totalLowPages + value
	}
	return float64(totalLowPages), nil
}

func (o *MeminfoObserver) estimateAvailableMemory(data map[string]float64, lowWatermarkPages float64) float64 {
	const bytesInKb = 1024

	memFreeKb := data["MemFree"]
	memActiveFileKb := data["Active(file)"]
	memInactiveFileKb := data["Inactive(file)"]
	memSReclaimableKb := data["SReclaimable"]

	memLowWatermarkKb := lowWatermarkPages * float64(o.pageSize) / bytesInKb

	memAvailableKb := memFreeKb - memLowWatermarkKb
	memPageCacheKb := memActiveFileKb + memInactiveFileKb

	memPageCacheKb -= math.Min(memPageCacheKb/2, memLowWatermarkKb)
	memAvailableKb = memAvailableKb + memPageCacheKb

	memSReclaimableKb -= math.Min(memSReclaimableKb/2, memLowWatermarkKb)
	memAvailableKb = memAvailableKb + memSReclaimableKb

	return memAvailableKb
}

func (o *MeminfoObserver) analyze() (map[string]interface{}, error) {
	const totalKey string = "mem_total"
	const availableKey string = "mem_avail"
	const availableEstimatedKey string = "mem_avail_est"
	const percentKey string = "mem_pcnt"
	const swapPercentKey string = "swp_pcnt"
	const swapFreeKey string = "swp_free"
	const swapTotalKey string = "swp_total"
	const reclaimableKey string = "mem_reclaim"
	const inactiveFileKey string = "mem_inactive"

	const bytesInKb = 1024

	result := make(map[string]interface{})
	memInfoData, err := o.reader.getFloatKeyValuePairs(meminfoFile)
	if err != nil {
		return nil, err
	}

	lowWatermarkPages, err := o.getLowPages()
	if err != nil {
		return nil, err
	}
	memAvailableEstimatedKb := o.estimateAvailableMemory(memInfoData, lowWatermarkPages)

	memAvailableKb, ok := memInfoData["MemAvailable"]
	if ok {
		result[availableEstimatedKey] = memAvailableEstimatedKb / bytesInKb
		result[availableKey] = memAvailableKb / bytesInKb
	} else {
		// No 'MemAvailable' key in 'meminfo', looks like we're running an old kernel
		memAvailableKb = memAvailableEstimatedKb
		result[availableKey] = memAvailableEstimatedKb / bytesInKb
	}

	if o.params["showReclaimable"] == "true" {
		memReclaimableKb, ok := memInfoData["SReclaimable"]
		if ok {
			result[reclaimableKey] = memReclaimableKb / bytesInKb
		}
	}

	if o.params["showInactive"] == "true" {
		memInaciveFileKb, ok := memInfoData["Inactive(file)"]
		if ok {
			result[inactiveFileKey] = memInaciveFileKb / bytesInKb
		}
	}

	memTotalKb, ok := memInfoData["MemTotal"]
	result[totalKey] = memTotalKb / bytesInKb

	percent := (memTotalKb - memAvailableKb) * 100 / memTotalKb
	result[percentKey] = float64(percent)

	swapTotalKb, ok := memInfoData["SwapTotal"]
	if swapTotalKb > 0 {
		result[swapTotalKey] = swapTotalKb / bytesInKb
		swapFreeKb, _ := memInfoData["SwapFree"]
		result[swapFreeKey] = swapFreeKb / bytesInKb
		swapPercent := (swapTotalKb - swapFreeKb) * 100 / swapTotalKb
		result[swapPercentKey] = float64(swapPercent)
	}

	return result, nil
}

func (o *MeminfoObserver) process() {
	actualData, err := o.analyze()
	if err != nil {
		log.Fatal(err)
	} else {
		o.tracker.track(&actualData)
	}
}
