package main

import "math"
import "testing"

func floatsEqual(a float64, b float64) bool {
	epsilon := 0.01 // actually we don't care about the precision in this scenario
	diff := math.Abs(a - b)
	return diff < epsilon
}

type sampleData struct {
	deltaTime    float64
	deltaFaults  int64
	expectedEwma float64
}

func TestCalculateSwapFaultsSimple(t *testing.T) {
	var ewma = 100.0
	const halfLife = 100
	o := SwapObserver{nil, nil, 1000, swapFaultsValues{0, ewma, 0, 0, 0}, halfLife, false}
	step1results := o.calculateSwapFaults(1, halfLife)
	if !floatsEqual(step1results.currentFaultsPerSecond, ewma/2) {
		t.Fatalf("Wrong step1 EWMA page faults calculations: expected %f, got %f", ewma/2, step1results.currentFaultsPerSecond)
	}
}

func TestCalculateSwapFaultsNullDelta(t *testing.T) {
	var ewma = 100.0
	const halfLife = 100
	o := SwapObserver{nil, nil, 1000, swapFaultsValues{0, ewma, 0, 0, 0}, halfLife, false}
	step1results := o.calculateSwapFaults(1, halfLife)
	o.oldValues = step1results
	step2results := o.calculateSwapFaults(2, halfLife)
	if !floatsEqual(step2results.currentFaultsPerSecond, step1results.currentFaultsPerSecond) {
		t.Fatalf("Wrong null delta EWMA page faults calculations: expected %f, got %f", step1results.currentFaultsPerSecond, step2results.currentFaultsPerSecond)
	}
}

func TestCalculateSwapFaultStepped(t *testing.T) {
	var ewma = 100.0
	const halfLife = 100
	o := SwapObserver{nil, nil, 1000, swapFaultsValues{0, ewma, 0, 0, 0}, halfLife, false}
	const increments = 10
	var step2results swapFaultsValues
	for i := 1; i <= increments; i++ {
		step2results = o.calculateSwapFaults(1, float64(halfLife*i)/float64(increments))
	}
	if !floatsEqual(step2results.currentFaultsPerSecond, ewma/2) {
		t.Fatalf("Wrong step2 EWMA page faults calculations: expected %f, got %f", ewma/2, step2results.currentFaultsPerSecond)
	}
}

func TestCalculateSwapFaultPrecalc(t *testing.T) {
	var step3results swapFaultsValues
	var cpuTime float64 = 1.0
	var pageFaults int64 = 1
	const halfLifeStep3 = 1
	o := SwapObserver{nil, nil, 1000, swapFaultsValues{float64(pageFaults), 0.0, pageFaults, cpuTime, 0}, halfLifeStep3, false}
	samples := []sampleData{
		sampleData{1, 10, 5.0},
		sampleData{1, 10, 7.5},
		sampleData{2, 0, 1.875},
		sampleData{3, 420, 122.734375},
		sampleData{1, 24, 73.3671875},
	}
	for i, sample := range samples {
		cpuTime += sample.deltaTime
		pageFaults += sample.deltaFaults
		step3results = o.calculateSwapFaults(pageFaults, cpuTime)
		o.oldValues = step3results
		if !floatsEqual(step3results.currentFaultsPerSecond, sample.expectedEwma) {
			t.Fatalf("Wrong step3 iteration #%d EWMA page faults calculations: expected %f, got %f", i, sample.expectedEwma, step3results.currentFaultsPerSecond)
		}
	}
}
