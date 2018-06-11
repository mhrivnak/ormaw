package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"

	"k8s.io/api/admission/v1beta1"
)

type Request struct {
	UID string
}

type ReqBody struct {
	Request Request
}

type RespWrapper struct {
	Kind       string `json:"kind"`
	ApiVersion string `json:"apiVersion"`
	Response   AResp  `json:"response"`
}

type AResp struct {
	UID     string `json:"uid"`
	Allowed bool   `json:"allowed"`
}

func dumper(w http.ResponseWriter, r *http.Request) {
	dump, err := httputil.DumpRequest(r, true)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(dump))

	defer r.Body.Close()
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}

	aReview := v1beta1.AdmissionReview{}

	//areq := ReqBody{}
	err = json.Unmarshal(data, &aReview)
	if err != nil {
		panic(err)
	}

	w.Header().Set("content-type", "application/json")

	wrapper := v1beta1.AdmissionReview{
		Response: &v1beta1.AdmissionResponse{UID: aReview.Request.UID, Allowed: true},
	}

	respBody, err := json.Marshal(wrapper)
	if err != nil {
		panic(err)
	}
	w.Write(respBody)
}

func main() {
	http.HandleFunc("/", dumper)
	err := http.ListenAndServeTLS(":4443", "certs/yyc.crt", "certs/yyc.key", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
