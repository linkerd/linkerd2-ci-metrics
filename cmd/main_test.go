package main

import (
	"encoding/json"
	"io/ioutil"
	"testing"
)

func TestProcessData(t *testing.T) {
	var jobs []JobRun
	var annotations []ErrorAnn
	data1, _ := ioutil.ReadFile("testdata/jobs.json")
	if err := json.Unmarshal(data1, &jobs); err != nil {
		t.Fatal(err)
	}
	data2, _ := ioutil.ReadFile("testdata/annotations.json")
	if err := json.Unmarshal(data2, &annotations); err != nil {
		t.Fatal(err)
	}
	if err := processData(jobs, annotations); err != nil {
		t.Fatal(err)
	}
}
