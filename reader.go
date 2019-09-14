package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
)

type Reader interface {
	getSumAllIntValues(filename string, key string) ([]int64, error)
	getTextValue(filename string, key string) (string, error)
	getFloatValue(filename string, key string) (float64, error)
	getIntValue(filename string, key string) (int64, error)
	getIntWhole(filename string) (int64, error)
	getFloatKeyValuePairs(filename string) (result map[string]float64, err error)
}

type FileReader struct {
}

func trimLastSemicolon(text string) string {
	if last := len(text) - 1; last >= 0 && text[last] == ':' {
		text = text[:last]
	}
	return text
}

func (o FileReader) getFloatKeyValuePairs(filename string) (result map[string]float64, err error) {
	result = make(map[string]float64)
	file, err := os.Open(filename)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		data := strings.Fields(scanner.Text())
		key := trimLastSemicolon(data[0])
		value, err := strconv.ParseFloat(data[1], 64)
		if err != nil {
			log.Print(fmt.Sprintf("Error '%s' while parsing '%s' in '%s'", err, key, filename))
		} else {
			result[key] = value
		}
	}
	err = scanner.Err()
	return
}

func (o FileReader) getSumAllIntValues(filename string, key string) ([]int64, error) {
	var result []int64
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		data := strings.Fields(scanner.Text())
		if data[0] == key {
			i, err := strconv.ParseInt(data[1], 10, 64)
			if err == nil {
				result = append(result, i)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (o FileReader) getTextValue(filename string, key string) (string, error) {
	var result string
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		data := strings.Fields(scanner.Text())
		if trimLastSemicolon(data[0]) == key {
			result = data[1]
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}

	if len(result) == 0 {
		err = fmt.Errorf("Key '%s' was not found in '%s'", key, filename)
		return "", err
	}
	return result, nil
}

func (o FileReader) getFloatValue(filename string, key string) (float64, error) {
	var result float64 = math.NaN()
	text, err := o.getTextValue(filename, key)
	if err == nil {
		result, err = strconv.ParseFloat(text, 64)
	}
	return result, err
}

func (o FileReader) getIntValue(filename string, key string) (int64, error) {
	var result int64
	text, err := o.getTextValue(filename, key)
	if err == nil {
		result, err = strconv.ParseInt(text, 10, 64)
	}
	return result, err
}

func (o FileReader) getIntWhole(filename string) (int64, error) {
	text, err := ioutil.ReadFile(filename)
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(strings.TrimSpace(string(text)), 10, 64)
}
