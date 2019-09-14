package main

import "reflect"
import "strings"
import "testing"

type FileReaderStub struct {
}

func (r FileReaderStub) generateFakePath(realPath string) string {
	const fakePathPrefix string = "./test_samples/"
	pathElements := strings.Split(realPath, "/")
	filename := pathElements[len(pathElements)-1]
	return fakePathPrefix + filename + ".txt"
}

func (r FileReaderStub) getSumAllIntValues(filename string, key string) ([]int64, error) {
	o := FileReader{}
	return o.getSumAllIntValues(r.generateFakePath(filename), key)
}

func (r FileReaderStub) getTextValue(filename string, key string) (string, error) {
	o := FileReader{}
	return o.getTextValue(r.generateFakePath(filename), key)
}

func (r FileReaderStub) getFloatValue(filename string, key string) (float64, error) {
	o := FileReader{}
	return o.getFloatValue(r.generateFakePath(filename), key)
}

func (r FileReaderStub) getIntValue(filename string, key string) (int64, error) {
	o := FileReader{}
	return o.getIntValue(r.generateFakePath(filename), key)
}

func (r FileReaderStub) getIntWhole(filename string) (int64, error) {
	o := FileReader{}
	return o.getIntWhole(r.generateFakePath(filename))
}

func (r FileReaderStub) getFloatKeyValuePairs(filename string) (result map[string]float64, err error) {
	o := FileReader{}
	return o.getFloatKeyValuePairs(r.generateFakePath(filename))
}

func TestGetKeyValuePairs(t *testing.T) {
	r := FileReaderStub{}
	fetchedData, err := r.getFloatKeyValuePairs(meminfoFile)
	if err != nil {
		t.Fatal(err)
	}
	const expectedMemAvail = 26190788
	fetchedMemAvail := uint64(fetchedData["MemAvailable"])
	if fetchedMemAvail != expectedMemAvail {
		t.Errorf("'MemAvailable' param read incorrectly, expected %d, got %d", expectedMemAvail, fetchedMemAvail)
	}

	const expectedSwapCached = 0
	fetchedSwapCached := uint64(fetchedData["SwapCached"])
	if fetchedSwapCached != expectedSwapCached {
		t.Errorf("'SwapCached' param read incorrectly, expected %d, got %d", expectedSwapCached, fetchedSwapCached)
	}

	const expectedDirect = 16777216
	fetchedDirect := uint64(fetchedData["DirectMap1G"])
	if fetchedDirect != expectedDirect {
		t.Errorf("'DirectMap1G' param read incorrectly, expected %d, got %d", expectedDirect, fetchedDirect)
	}
}

func TestGetTextValue(t *testing.T) {
	r := FileReaderStub{}
	fetchedData, err := r.getTextValue(psiMemoryFile, "some")
	if err != nil {
		t.Fatal(err)
	}
	const expectedSomeString = "avg10=0.00"
	if fetchedData != expectedSomeString {
		t.Errorf("'some' text value parsed incorrectly, expected '%s', got '%s'", expectedSomeString, fetchedData)
	}
}

func TestGetAllIntValues(t *testing.T) {
	r := FileReaderStub{}
	fetchedData, err := r.getSumAllIntValues(zoneFile, "low")
	if err != nil {
		t.Fatal(err)
	}
	var expectedData = []int64{11, 2656, 67067, 0, 0}
	if !reflect.DeepEqual(fetchedData, expectedData) {
		t.Logf("Expected result: %#v", expectedData)
		t.Logf("Got result: %#v", fetchedData)
		t.Fail()
	}
}
