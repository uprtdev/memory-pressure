package main

/*
#include <unistd.h>
*/
import "C"
import (
	"fmt"
	"log"
	"math"
	"strconv"
)

const vmstatPath = "/proc/vmstat"
const meminfoPath = "/proc/meminfo"
const swapinessPath = "/proc/sys/vm/swappiness"

type SwapObserver struct {
	tracker                *Tracker
	reader                 Reader
	params                 map[string]string
	hertz                  uint
	oldValues              swapFaultsValues
	lowPassHalfLifeSeconds float64
}

type swapFaultsValues struct {
	sampledFaultsPerSecond float64
	currentFaultsPerSecond float64
	lastMajorPageFaults    int64
	lastUserTime           float64
	multiplier             float64
}

func (o *SwapObserver) Initialize(t *Tracker, r Reader, p map[string]string) {
	o.tracker = t
	o.reader = r
	o.params = p
	o.hertz = uint(C.sysconf(C._SC_CLK_TCK))
	log.Printf("System timer frequency is %d Hz", o.hertz)
	o.oldValues.lastUserTime = 0
	lowPassOptionStr := o.params["lowPassHalfLifeSeconds"]
	if len(lowPassOptionStr) > 0 {
		if f, err := strconv.ParseFloat(lowPassOptionStr, 32); err == nil {
			o.lowPassHalfLifeSeconds = f
			log.Printf("Using lowPassHalfLife = %f seconds", f)
		}
	} else {
		o.lowPassHalfLifeSeconds = 30.0
	}

	o.process()
}

func (o *SwapObserver) TimerEvent() {
	o.process()
}

func (o *SwapObserver) getUserTime() (float64, error) {
	if o.hertz == 0 {
		return 0, fmt.Errorf("Failed to get system ticks frequency")
	}
	timeSinceBoot, err := o.reader.getIntValue("/proc/stat", "cpu")
	if err != nil {
		return 0, err
	}
	return float64(timeSinceBoot) / float64(o.hertz), nil
}

func (o *SwapObserver) getValuesForTendency() (nrMapped int64, memTotal int64, swappiness int64, err error) {
	nrMapped, err = o.reader.getIntValue(vmstatPath, "nr_mapped")
	if err != nil {
		return
	}

	memTotalKb, err := o.reader.getIntValue(meminfoPath, "MemTotal")
	if err != nil {
		return
	}
	memTotal = memTotalKb * 1024

	swappiness, err = o.reader.getIntWhole(swapinessPath)
	return
}

func (o *SwapObserver) getValuesForFaults() (newMajorPageFaults int64, newUserExecTime float64, err error) {
	newMajorPageFaults, err = o.reader.getIntValue(vmstatPath, "pgmajfault")
	if err != nil {
		return
	}
	newUserExecTime, err = o.getUserTime()
	return
}

func (o *SwapObserver) calculateTendency(nrMappedValue int64, memTotal int64, swappiness int64) (float64, error) {
	mappedRatio := float64(nrMappedValue*int64(PageSize())*100) / float64(memTotal)
	swapTendency := mappedRatio/2 + float64(swappiness)
	return swapTendency, nil
}

func (o *SwapObserver) calculateSwapFaults(newMajorPageFaults int64, newUserExecTime float64) swapFaultsValues {
	var values swapFaultsValues

	deltaUserExecTime := newUserExecTime - o.oldValues.lastUserTime
	deltaMajorPageFaults :=
		float64(newMajorPageFaults - o.oldValues.lastMajorPageFaults)

	if o.oldValues.lastUserTime == 0 && o.params["averageOnlyCurrent"] == "true" {
		// If we have an option to calculate current average relying only on new measures
		// and this is our first run, let's skip this calculation
		deltaUserExecTime = 0
	}

	if deltaUserExecTime > 0 {
		values.sampledFaultsPerSecond =
			deltaMajorPageFaults / deltaUserExecTime
		adjustedEwmaCoefficient :=
			1 - math.Exp2(-deltaUserExecTime/o.lowPassHalfLifeSeconds)
		values.currentFaultsPerSecond =
			adjustedEwmaCoefficient*values.sampledFaultsPerSecond +
				(1-adjustedEwmaCoefficient)*o.oldValues.currentFaultsPerSecond
	} else {
		values.sampledFaultsPerSecond = o.oldValues.sampledFaultsPerSecond
		values.currentFaultsPerSecond = o.oldValues.currentFaultsPerSecond
	}

	values.lastUserTime = newUserExecTime
	values.lastMajorPageFaults = newMajorPageFaults

	averageFaultsPerSecond := float64(newMajorPageFaults) / float64(newUserExecTime)
	if averageFaultsPerSecond > 0 {
		values.multiplier = values.currentFaultsPerSecond / averageFaultsPerSecond
	} else {
		values.multiplier = 0
	}

	return values
}

func (o *SwapObserver) analyze() (map[string]interface{}, error) {
	const pgmajfaultKey string = "swp_flts"
	const faultsSecKey string = "swp_flts_sec"
	const tendencyKey string = "swp_tend"
	const faultsSecFilterKey string = "swp_flts_sec_f"
	const faultsMultiplierKey string = "swp_flts_mult"

	newMajorPageFaults, newUserExecTime, err := o.getValuesForFaults()
	if err != nil {
		return nil, err
	}

	newValues := o.calculateSwapFaults(newMajorPageFaults, newUserExecTime)
	o.oldValues = newValues

	result := make(map[string]interface{})
	result[faultsSecKey] = newValues.sampledFaultsPerSecond
	result[faultsSecFilterKey] = newValues.currentFaultsPerSecond
	result[faultsMultiplierKey] = newValues.multiplier

	nrMappedValue, memTotalValue, swappiness, err := o.getValuesForTendency()
	if err == nil {
		tendency, err := o.calculateTendency(nrMappedValue, memTotalValue, swappiness)
		if err == nil {
			result[tendencyKey] = tendency
		}
	}
	if err != nil {
		log.Print(err)
	}
	return result, nil
}

func (o *SwapObserver) process() {
	actualData, err := o.analyze()
	if err != nil {
		log.Fatal(err)
	} else {
		o.tracker.track(&actualData)
	}
}
