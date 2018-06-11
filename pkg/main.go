package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"

	"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// patch - represents a JSONPatch
type patch struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value"`
}

// dumper - dumps the incoming request to stdout and also creates a patch for
// the OwnerReferences
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

	err = json.Unmarshal(data, &aReview)
	if err != nil {
		panic(err)
	}

	w.Header().Set("content-type", "application/json")

	pt := v1beta1.PatchTypeJSONPatch
	sp, err := makePatch(&aReview)
	if err != nil {
		panic(err)
	}

	wrapper := v1beta1.AdmissionReview{
		Response: &v1beta1.AdmissionResponse{
			UID:       aReview.Request.UID,
			Allowed:   true,
			Patch:     sp,
			PatchType: &pt,
		},
	}

	respBody, err := json.Marshal(wrapper)
	if err != nil {
		panic(err)
	}
	w.Write(respBody)
}

// makePatch - creates and returns a patch, encoded as a JSON byte array, based
// on the passed-in review.
func makePatch(aReview *v1beta1.AdmissionReview) ([]byte, error) {
	owner := metav1.OwnerReference{
		APIVersion: "v1alpha1",
		Kind:       "Foo",
		Name:       "example-foo",
		UID:        types.UID("97388dad-6da8-11e8-a81d-847297420507"),
	}

	p := patch{
		Op:    "add",
		Path:  "/metadata/OwnerReferences",
		Value: []metav1.OwnerReference{owner},
	}
	sp, err := json.Marshal([]patch{p})
	if err != nil {
		return nil, err
	}
	fmt.Println(string(sp))
	return sp, nil
}

func main() {
	http.HandleFunc("/", dumper)
	err := http.ListenAndServeTLS(":4443", "certs/yyc.crt", "certs/yyc.key", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
