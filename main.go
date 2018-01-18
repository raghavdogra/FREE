package main
// #cgo pkg-config: opencv
// #cgo LDFLAGS: -Lcaffe/lib 
// #cgo CXXFLAGS: -std=c++11 -Icaffe/include -I/usr/include/opencv -I.. -O2 -fomit-frame-pointer -Wall
// #include <stdlib.h>
// #include "classification.hpp"
import "C"
import "unsafe"

import (
	"time"
        "io"
        "io/ioutil"
	"log"
	"net/http"
)

var cstr *C.char
var ctx *C.classifier_ctx
var requestCount uint8
var bigbuffer [] byte
var w1,w2 http.ResponseWriter
var mux map[string]func(http.ResponseWriter, *http.Request)
var responseReady bool
func modclass(w http.ResponseWriter, r *http.Request) {
        if (requestCount == 3) {
		requestCount  = 0
	}
	requestCount = requestCount + 1
	log.Println ("req count is ")
	log.Println (requestCount)
	buffer, err := ioutil.ReadAll(r.Body)
        if err != nil {
                http.Error(w, err.Error(), http.StatusBadRequest)
        return
        }

	if requestCount == 1 {
                responseReady = false
		w1 = w
		bigbuffer = append(bigbuffer,buffer...)
		for requestCount == 1 {

		}
	       cstr, err = C.classifier_classify( (*C.char)(unsafe.Pointer(&buffer[0])), C.size_t(len(buffer)))	
               responseReady = true
               io.WriteString(w1, C.GoString(cstr))
	} else if requestCount == 2 {
		w2 = w
		bigbuffer = append(bigbuffer,buffer...)
		log.Println("Count is two")
                for responseReady==false {

                }
		io.WriteString(w2,C.GoString(cstr))
		requestCount = 3
		bigbuffer = nil
	} else {}

}

type myHandler struct{}

func (*myHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h, ok := mux[r.URL.String()]; ok {
		h(w, r)
		return
	}

	io.WriteString(w, "My server: "+r.URL.String())
}

/*
func classify(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "", http.StatusMethodNotAllowed)
		return
	}
	buffer, err := ioutil.ReadAll(r.Body)
	cstr, err := C.classifier_classify( (*C.char)(unsafe.Pointer(&buffer[0])), C.size_t(len(buffer)))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	return
	}
	log.Println(C.GoString(cstr))
//	defer C.free(unsafe.Pointer(cstr))
	io.WriteString(w, C.GoString(cstr))
}
*/
func main() {
//	cmodel := C.CString(os.Args[1])
//	ctrained := C.CString(os.Args[2])
//	cmean := C.CString(os.Args[3])
//	clabel := C.CString(os.Args[4])
//
//	log.Println(os.Args[1])
//	log.Println(os.Args[2])
//	log.Println(os.Args[3])
//	log.Println(os.Args[4])
	srv := http.Server{
		Addr:    ":8000",
		ReadTimeout: 50000 * time.Second,
		WriteTimeout: 10000 * time.Second,
		Handler: &myHandler{},
	}
	mux = make(map[string]func(http.ResponseWriter, *http.Request))
	mux["/api/classify"] = modclass

	log.Println("Initializing Caffe classifiers")
	ctx, err := C.classifier_initialize()
        if err != nil {
                log.Fatalln("could not initialize classifier:", err)
                return
        }
	bigbuffer = nil
	requestCount = 0
//	log.Println((ctx))
	defer C.classifier_destroy(ctx)
	log.Println("Adding REST endpoint /api/classify")
	log.Println("Starting server listening on :8000")
	log.Fatal(srv.ListenAndServe())
 }


