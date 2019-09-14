package main

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

type Tracker struct {
	mtx       sync.Mutex
	actual    map[string]interface{}
	fields    []string
	startTime int64
}

func (t *Tracker) prepare() {
	t.startTime = time.Now().Unix()
	t.saveTime()
}

func (t *Tracker) track(update *map[string]interface{}) {
	if t.actual == nil {
		t.actual = make(map[string]interface{})
	}
	for k, v := range *update {
		t.trackOne(k, v)
	}
}

func (t *Tracker) trackOne(key string, value interface{}) {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	if t.actual == nil {
		t.actual = make(map[string]interface{})
	}

	t.actual[key] = value
}

func (t *Tracker) prepareAndPrintHeader() {
	t.fields = make([]string, 0)
	for k := range t.actual {
		t.fields = append(t.fields, k)
	}
	sort.Strings(t.fields)
	for _, j := range t.fields {
		fmt.Printf("%*s, ", len(j), j)
	}
	fmt.Println("")
}

func (t *Tracker) printData() {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	for n, k := range t.fields {
		switch t.actual[k].(type) {
		default:
			fmt.Printf("%*v, ", len(t.fields[n]), t.actual[k])
		case float64:
			fmt.Printf("%*.2f, ", len(t.fields[n]), t.actual[k].(float64))
		}
	}
	fmt.Println("")
}

func (t *Tracker) saveTime() {
	actualTime := time.Now().Unix() - t.startTime
	t.trackOne("time", actualTime)
}
