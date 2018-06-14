package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"
	"strings"

	"k8s.io/api/admission/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// patch - represents a JSONPatch
type patch struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value"`
}

type session struct {
	Clientset  *kubernetes.Clientset
	TargetType string
}

// dumper - dumps the incoming request to stdout and also creates a patch for
// the OwnerReferences
func (s *session) dumper(w http.ResponseWriter, r *http.Request) {
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

	owner, err := s.getOwner(&aReview)
	if err != nil {
		panic(err)
	}

	wrapper := v1beta1.AdmissionReview{
		Response: &v1beta1.AdmissionResponse{
			UID:     aReview.Request.UID,
			Allowed: true,
		},
	}

	// If an owner reference was found on the ServiceAccount, add a patch
	if owner != nil {
		sp, err := makePatch(owner)
		if err != nil {
			panic(err)
		}

		pt := v1beta1.PatchTypeJSONPatch
		wrapper.Response.Patch = sp
		wrapper.Response.PatchType = &pt
	}

	respBody, err := json.Marshal(wrapper)
	if err != nil {
		panic(err)
	}
	w.Header().Set("content-type", "application/json")
	w.Write(respBody)
}

func (s *session) getOwner(aReview *v1beta1.AdmissionReview) (*metav1.OwnerReference, error) {
	api := s.Clientset.CoreV1()
	parts := strings.Split(aReview.Request.UserInfo.Username, ":")
	// If it's a ServiceAccount, the username will be in the form
	// "system:serviceaccount:<namespace>:<name>"
	if len(parts) == 4 && parts[0] == "system" && parts[1] == "serviceaccount" {
		user, err := api.ServiceAccounts(parts[2]).Get(parts[3], metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		fmt.Printf("%v\n", user)

		for _, oref := range user.OwnerReferences {
			if oref.Kind == s.TargetType {
				return &oref, nil
			}
		}
	}
	return nil, nil
}

// makePatch - creates and returns a patch, encoded as a JSON byte array, based
// on the passed-in review. It will overwrite any existing OwnerReference.
func makePatch(owner *metav1.OwnerReference) ([]byte, error) {
	p := patch{
		Op:    "add",
		Path:  "/metadata/OwnerReferences",
		Value: []metav1.OwnerReference{*owner},
	}
	sp, err := json.Marshal([]patch{p})
	if err != nil {
		return nil, err
	}
	fmt.Println(string(sp))
	return sp, nil
}

func main() {
	// Setup an API connection in a "session"
	home := os.Getenv("HOME")
	configpath := filepath.Join(home, ".kube/config")
	config, err := clientcmd.BuildConfigFromFlags("", configpath)
	if err != nil {
		log.Fatal("BuildConfigFromFlags: ", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal("NewForConfig: ", err)
	}
	session := session{Clientset: clientset, TargetType: "Foo"}

	// Serve the webhook endpoint
	http.HandleFunc("/", session.dumper)
	err = http.ListenAndServeTLS(":4443", "certs/yyc.crt", "certs/yyc.key", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
