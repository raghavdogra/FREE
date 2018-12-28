package main
// #include <stdlib.h>
// #include <zmq.h>
// #include "classification.hpp"
// #cgo windows LDFLAGS: -L/usr/local/lib -lzmq
import "C"

import (
//	"bytes"
	"bytes"
	"encoding/binary"
	"time"
        "io"
       // "io/ioutil"
	"log"
	"os"
	"net/http"
	"strconv"
//	"image"
	"image/jpeg"
	zmq "github.com/pebbe/zmq4"
)
var n uint32
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

type pixel struct{
	r,g,b uint32
}

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
	batchsize int
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
		jobs:= [9]job{}
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
//		go processbatch(jobs,i)
		go argusBackend(jobs,i)
		}
	}
}
func stage2(j job, count int){
	log.Println(time.Duration(count)*time.Millisecond)
	time.Sleep(time.Duration(count)*time.Millisecond)
	j.ch <-"Rat"
}
func dummygpu() {
	gpu_channel = make(chan job, 1)
	job_num := 0
	for true {

			currjob := <-gpu_channel
			job_num = job_num + 1
			var latency int
			var throughput int
			log.Print("gpu request# ",job_num, "bSize: ",currjob.batchsize)
			switch  currjob.batchsize {
			case 1 :
				throughput = 10
				latency = 32
			case 2 :
				throughput = 11
				latency = 34
			case 3 :
				throughput = 12
				latency = 37
			case 4 :
				throughput = 13
				latency = 42
			case 5 :
				throughput = 14
				latency = 43
			case 6 :
				throughput = 15
				latency = 45
			case 7 :
				throughput = 17
				latency = 48
			case 8 :
				throughput = 18
				latency = 52
			case 9 :
				throughput = 20
				latency = 53
			}
			log.Println(time.Duration(throughput)*time.Millisecond)
			time.Sleep((time.Duration(throughput) * time.Millisecond))
			go stage2(currjob, latency-throughput)
	}
}

func argusBackend(jobs [9]job, count int) {
	context, _ := zmq.NewContext()
	//defer context.Close()
	zsock, _ := context.NewSocket(zmq.DEALER)
	defer zsock.Close()
	zsock.Connect("tcp://localhost:10090")
	var s []string
	s = make([]string, count, count)
	for i:=0;i<count;i++ {
		//zsock.Send(string(jobs[i].buf),Flag(C.ZMQ_SNDMORE))
		s[i] = string(jobs[i].buf)
	}
	zsock.SendMessage(s)
	msg, _ := zsock.RecvMessage(0)
//	log.Println("Received from Argus Backend")
//	log.Println(len(msg))
	for i:=0;i<count;i++ {
	//	zsock.Send(string(jobs[i].buf),Flag(C.ZMQ_SNDMORE))
	//	log.Println(len(msg[i]))
	//	log.Printf("type is %T",msg[i])
	var fres [1000]float32
	arr := [] byte(msg[i])
//	for i:=0 ; i < 4000 ;i++ {
//		log.Print(arr[i])
//	}
	buffferReceived := bytes.NewReader(arr)
	err := binary.Read(buffferReceived , binary.LittleEndian, &fres)
	if err != nil {
		log.Println("binary.Read failed:", err)
	}
	log.Println(fres[0])
		jobs[i].ch <- string(len(fres))
		//log.Println(msg[i])
	}
}

func processbatch(jobs [9]job, count int ) {
	buf1 := jobs[0].buf
	res_chan := make (chan string)
	log.Print("sending to gpu and count is ", count)
	gpu_channel <- job{res_chan,buf1,count}
	log.Print("waiting to recieve from res_chan")
	cstr := <-res_chan
	log.Print("recieved from res_chan")
	i:=0
        for i=0;i<count;i++ {
		jobs[i].ch <- cstr
	}
}

func modclass1(w http.ResponseWriter, r *http.Request) {
/*        buffer, err := ioutil.ReadAll(r.Body)
        if err != nil {
                http.Error(w, err.Error(), http.StatusBadRequest)
        return
        }
	log.Print(len(buffer))*/
	im, err := jpeg.Decode(r.Body)
        if err != nil {
		log.Print("Decode Failed, returning")
               http.Error(w, err.Error(), http.StatusBadRequest)
      		return
        }
//	bounds := im.Bounds()
	
//	log.Print(bounds)
/*
//	log.Print(len(string(im)))
//	m0 := image.NewRGBA(image.Rect(0, 0, 224, 224))
	imag := im.SubImage(image.Rect(0, 0, 224, 224)).(*image.RGBA)
	var buf bytes.Buffer
	err = jpeg.Encode(&buf, im, nil)
	if err != nil {
		log.Print("Encode Failed, returning")
               http.Error(w, err.Error(), http.StatusBadRequest)
		return 
	}
	log.Print(len(buf.Bytes()))
	//newImage := resize.Resize(160, 0, original_image, resize.Lanczos3)
	//log.Print(buff)*/
	pixels := make([]pixel, 224*224)
	pbuffer := make([]byte, 224*224*3*4)
	for i:=0; i < 224*224; i++ {
		x := i%224;
		y := i/224;
		r,g,b,_ := im.At(x,y).RGBA()
		pixels[i].r = r
		pixels[i].g = g
		pixels[i].b = b	
		*(&pbuffer[i*12 + 4]) = byte(g)
		*(&pbuffer[i*12]) = byte(r)
		*(&pbuffer[i*12 + 8]) = byte(b)
	}
	ch := make(chan string) //creates it's own channel from which it will expect to receive value back from GPU!
	c<-job{ch,pbuffer,0} //sends the job structure to c
	gostr := <-ch	//waits to receive from channel which it passed on that it'll receive from that!
	io.WriteString(w, gostr )
}

func main() {
	srv := http.Server{
		Addr:    ":8002",
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
	log.Println("Adding REST endpoint /api/classify")
	log.Println("Starting server listening on :8002 with SIM backend")
	log.Fatal(srv.ListenAndServe())
 }

