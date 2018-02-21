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
	"sync/atomic"
)
var n uint32
var cstr *C.char
var ctx *C.classifier_ctx
/*
type SafeCounter struct {
	requestCount uint32
	mux sync.Mutex
}*/
var requestCount uint32
var bigbuffer [] byte
var w1 http.ResponseWriter
var w2 [10]http.ResponseWriter
var mux map[string]func(http.ResponseWriter, *http.Request)
var responseReady bool

func modclass(w http.ResponseWriter, r *http.Request) {
        if (atomic.LoadUint32(&requestCount) > n) {
		//requestCount  = 0
		atomic.StoreUint32(&requestCount,0)
	}
	buffer, err := ioutil.ReadAll(r.Body)
        if err != nil {
                http.Error(w, err.Error(), http.StatusBadRequest)
        return
        }
	bigbuffer = append(bigbuffer,buffer...)
	//requestCount = requestCount + 1
	atomic.AddUint32(&requestCount,1)
//	log.Println ("req count is ")
	log.Println (requestCount)

//	if requestCount == 1 {
	if atomic.LoadUint32(&requestCount) == 1 {
                responseReady = false
	//	w1 = w
		//for requestCount < n {
		for requestCount < n {

		}
	       cstr, err = C.classifier_classify( (*C.char)(unsafe.Pointer(&buffer[0])), C.size_t(len(buffer)))	
               responseReady = true
               io.WriteString(w, C.GoString(cstr))
	} else if atomic.LoadUint32(&requestCount) <= n {
	//	w2 = w
                for responseReady==false {

                }
		io.WriteString(w,C.GoString(cstr))
		//if requestCount==n{
		if atomic.LoadUint32(&requestCount)==n{
		//requestCount = n + 1
		atomic.AddUint32(&requestCount,1)
		bigbuffer = nil}
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
//var c chan struct{*http.Request;chan string}

type job struct {
	ch chan string
	buf []byte 
}
var c chan job
var gpu_channel chan job
func mainloop() {
        n:=1
	j:=0
	c= make(chan job,n)
        //var jobs [n]job
	i:=0
	for true {
		jobs:= make([]job,n)
                for i=0;i<n;i++{
			jobs[i] = <-c
			j = j+1
			log.Println(j)
		}
	//	c1 := r1.ch
	//	c2 := r2.ch
		go processbatch(jobs,i)

	}
}
func dummygpu() {
	gpu_channel = make(chan job)
	job_num := 0
	for true {
		currjob := <-gpu_channel
		job_num = job_num + 1
		log.Print("gpu request# ",job_num)
		time.Sleep(3 * time.Millisecond)
		currjob.ch <- "Rat"
	}
}
func processbatch(jobs []job, count int ) {
	buf1 := jobs[0].buf
//	cstr, err := C.classifier_classify( (*C.char)(unsafe.Pointer(&buf1[0])), C.size_t(len(buf1)))
	res_chan := make (chan string)
	log.Print("sending to gpu and count is ", count)
	gpu_channel <- job{res_chan,buf1}
	log.Print("waiting to recieve from res_chan")
	cstr := <-res_chan
	log.Print("recieved from res_chan")
//	if err != nil {
//                cstr = C.CString("error")
//        }
	i:=0
        for i=0;i<count;i++ {
	log.Print("trying to send back")
		//jobs[i].ch <- C.GoString(cstr)
		jobs[0].ch <- cstr
	log.Print("value sent back to chan")
	}
}

func modclass1(w http.ResponseWriter, r *http.Request) {
        buffer, err := ioutil.ReadAll(r.Body)
        if err != nil {
                http.Error(w, err.Error(), http.StatusBadRequest)
        return
        }
	ch := make(chan string)
	c<-job{ch,buffer}
	gostr := <-ch
	log.Println("I'm done!")
	io.WriteString(w, gostr )
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
		ReadTimeout: 10 * time.Second,
		WriteTimeout: 10 * time.Second,
		Handler: &myHandler{},
	}
	mux = make(map[string]func(http.ResponseWriter, *http.Request))
	mux["/api/classify"] = modclass1

	log.Println("Initializing Caffe classifiers")
//	ctx, err := C.classifier_initialize()
//        if err != nil {
//                log.Fatalln("could not initialize classifier:", err)
//                return
//        }
	bigbuffer = nil
	requestCount = 0
	n = 1
//	log.Println((ctx))
	go mainloop()
	go dummygpu()
	defer C.classifier_destroy(ctx)
	log.Println("Adding REST endpoint /api/classify")
	log.Println("Starting server listening on :8000")
	log.Fatal(srv.ListenAndServe())
 }

