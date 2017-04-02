/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package vpa

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"reflect"
	"testing"
)

const (
	protocol        = "http://"
	fakeAddress     = "localhost:8989"
	fakeHandlerName = "/echo"
)

func spinOffFakeRecommenderServer() (io.Closer, error) {
	mux := http.NewServeMux()

	mux.HandleFunc(fakeHandlerName, func(w http.ResponseWriter, r *http.Request) { // echo server
		body, _ := ioutil.ReadAll(r.Body)
		w.Write(body)
	})

	server := &http.Server{Addr: fakeAddress, Handler: mux}
	listener, err := net.Listen("tcp", fakeAddress) //created manually, to be able to close server later
	if err != nil {
		return nil, err
	}

	go server.Serve(listener)

	return listener, nil
}

func createFakeHTTPRecommenderClient() *httpJSONClient {
	client := &httpJSONClient{url: protocol + fakeAddress + fakeHandlerName}
	return client
}

func createFakeRecommenderClient() JSONClient {
	client := CreateRecommenderClient(protocol + fakeAddress + fakeHandlerName)
	return client
}

func TestSendJSON(t *testing.T) {
	closer, err := spinOffFakeRecommenderServer()
	if err != nil {
		t.Fatalf("Unable to create server %s", err.Error())
	}
	defer closer.Close()

	type SampleObject struct {
		Name string
		Body string
		Time int64
	}

	obj := SampleObject{"Alice", "Hello", 1294706395881547000}
	client := createFakeRecommenderClient()

	response, err := client.SendJSON(obj)
	if err != nil {
		t.Fatalf("Unable to send JSON: %s", err.Error())
	}
	var returnedObj SampleObject
	err = json.Unmarshal(response, &returnedObj)
	if err != nil {
		t.Fatalf("Unable to unmarshal JSON '%s' because of error: %s", string(response), err.Error())
	}

	if !reflect.DeepEqual(obj, returnedObj) {
		t.Errorf("Returned object: %+v do not match object which was sent: %+v ", returnedObj, obj)
	}
}

func TestSendData(t *testing.T) {
	closer, err := spinOffFakeRecommenderServer()
	if err != nil {
		t.Fatalf("Unable to create server %s", err.Error())
	}
	defer closer.Close()

	const requestBody = "fake request body"
	requestData := []byte(requestBody)

	client := createFakeHTTPRecommenderClient()
	resp, err := client.sendData(requestData, "plain/text")

	if err != nil {
		t.Fatalf("Unable to process rqquest %s", err.Error())
	}

	if string(resp) != string(requestData) {
		t.Errorf("Request body '%s' do not match response '%s'", requestData, resp)
	}
}
