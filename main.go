package main
// #cgo pkg-config: opencv
// #cgo LDFLAGS: -Lcaffe/lib 
// #cgo CXXFLAGS: -std=c++11 -Icaffe/include -I/usr/include/opencv -I.. -O2 -fomit-frame-pointer -Wall
// #include <stdlib.h>
// #include "classification.hpp"
import "C"

import (
	"time"
        "io"
        "io/ioutil"
	"log"
	"os"
	"net/http"
	"strconv"
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


type myHandler struct{}

func (*myHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h, ok := mux[r.URL.String()]; ok {
		h(w, r)
		return
	}

	io.WriteString(w, "My server: "+r.URL.String())
}

type job struct {
	ch chan string
	buf []byte 
}
var c chan job	//c is the channel from which mainloop receives forever.i
var gpu_channel chan job
func mainloop() {
        n,err:=strconv.Atoi(os.Args[1]) //the app will take the first arguement as the batch size
	if err!=nil {
		log.Print("error")
	}
        t,er1:=strconv.Atoi(os.Args[2]) //the app will take the first arguement as the batch size
	if er1!=nil {
		log.Print("error")
	}
	j:=0
	c= make(chan job)
        //var jobs [n]job
	i:=0
	for  {
		tick := time.Tick(time.Duration(t) * time.Millisecond)
		jobs:= [10]job{}
                for i=0;i<n;i++{
			select {
			case	jobs[i] = <-c:
				j = j+1
				log.Println(j)
				continue
			case <- tick:
				if i==0 {
//				log.Println("tick exploded")
				i = -1
				continue
				}
			}
			break
		}
		if i!=0 {
		go processbatch(jobs,i)
		}
	}
}
func stage2(j job){
	time.Sleep(9*time.Millisecond)
	j.ch <-"Rat"
}
func dummygpu() {
	gpu_channel = make(chan job, 4)
	job_num := 0
	for true {

			currjob := <-gpu_channel
			job_num = job_num + 1
			log.Print("gpu request# ",job_num)
			time.Sleep(3 * time.Millisecond)
			go stage2(currjob)
	}
}
func processbatch(jobs [10]job, count int ) {
	buf1 := jobs[0].buf
	res_chan := make (chan string)
	log.Print("sending to gpu and count is ", count)
	gpu_channel <- job{res_chan,buf1}
	log.Print("waiting to recieve from res_chan")
	cstr := <-res_chan
	log.Print("recieved from res_chan")
	i:=0
        for i=0;i<count;i++ {
		jobs[i].ch <- cstr
	}
}

func modclass1(w http.ResponseWriter, r *http.Request) {
        buffer, err := ioutil.ReadAll(r.Body)
        if err != nil {
                http.Error(w, err.Error(), http.StatusBadRequest)
        return
        }
	ch := make(chan string) //creates it's own channel from which it will expect to receive value back from GPU!
	c<-job{ch,buffer} //sends the job structure to c
	gostr := <-ch	//waits to receive from channel which it passed on that it'll receive from that!
	io.WriteString(w, gostr )
}

func main() {
	srv := http.Server{
		Addr:    ":8001",
		ReadTimeout: 100 * time.Second,
		WriteTimeout: 100 * time.Second,
		Handler: &myHandler{},
	}
	mux = make(map[string]func(http.ResponseWriter, *http.Request))
	mux["/api/classify"] = modclass1

	log.Println("Initializing Caffe classifiers")
	requestCount = 0
	n = 1
	go mainloop()//starts the main loop which receives the requests and batches them 
	go dummygpu()//concurrent dummy gpu running
	defer C.classifier_destroy(ctx)
	log.Println("Adding REST endpoint /api/classify")
	log.Println("Starting server listening on :8000")
	log.Fatal(srv.ListenAndServe())
 }

