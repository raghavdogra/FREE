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
	"sync"
        "io"
       // "io/ioutil"
	"log"
	"os"
	"net/http"
	"strconv"
	//"image"
	"image/jpeg"
	zmq "github.com/pebbe/zmq4"
)
var n uint32
var tc int
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
var totalRequests uint64
var totalBatches uint64
var mut sync.Mutex
func reportAvgBatchSize() {
	ticker := time.NewTicker(60*time.Second)
	for true {
		select {
			case <-ticker.C:
			mut.Lock()
			log.Println("Total Reqs: ",totalRequests,"Total Batches: ",totalBatches,"Avergae batch size: ",(float64(totalRequests)/float64(totalBatches)))
			totalRequests = 0
			totalBatches = 0
			mut.Unlock()
		}
	}
}

var c chan job	//c is the channel from which mainloop receives forever.i
var gpu_channel chan job

func getDesiredBatchSizeP100(avgRI int) int{
	reqRate := float64(1000.0/float64(avgRI))
	bs := 1
	if reqRate<95 {
		bs = 1
	} else if reqRate < 180 {
		bs = 2
	} else if reqRate < 250 {
		bs = 3
	} else if reqRate < 300 {
		bs = 4
	} else if reqRate < 330 {
		bs = 5
	} else if reqRate < 380 {
		bs = 6
	} else if reqRate < 400 {
		bs = 7
	} else if reqRate < 410 {
		bs = 8
	} else {
		bs = 9
	}
	return bs
}

func getDesiredBatchSizeK40(avgRI int) int{
	reqRate := float64(1000.0/float64(avgRI))
	bs := 1
	if reqRate<50 {
		bs = 1
	} else if reqRate < 70 {
		bs = 2
	} else if reqRate < 85 {
		bs = 3
	} else if reqRate < 90 {
		bs = 4
	} else if reqRate < 95 {
		bs = 5
	} else if reqRate < 100 {
		bs = 6
	} else if reqRate < 105 {
		bs = 7
	} else {
		bs = 8
	}
	return bs
}


func mainloop() {
        n,err:=strconv.Atoi(os.Args[1]) //the app will take the first arguement as the batch size
	if err!=nil {
		log.Print("error")
	}
        t,er1:=strconv.Atoi(os.Args[2]) //the app will take the first arguement as the batch size
	if er1!=nil {
		log.Print("error")
	}
		log.Print(n)
	j:=0
	c= make(chan job)
        //var jobs [n]job
	first:= [9]job{}
	first[0] = <-c
	i:=0
	start := time.Now()
	tick := time.Tick(time.Duration(t) * time.Millisecond)
	select {
        	case    first[1] = <-c:
			i = 2
        	case <- tick:
			i = 1
        }

	latest := time.Now()
	avgRI := int(latest.Sub(start))/1000000
	avgRI = max(avgRI,1)
	log.Println("first sample =")
	totalRequests = 0
	totalBatches = 0
	go processbatch(first,i)
//	go reportAvgBatchSize()
	i = 0
	for  {
		jobs:= [9]job{}
		bs:= getDesiredBatchSizeP100(int(avgRI))
		//log.Println("desired current batch size = ",bs)
		//log.Println("current avgRI = ",avgRI)
		jobs[0] = <-c
                avgRI = (9 * avgRI + 1 * (int(time.Since(latest))/1000000))/10
                avgRI = max(avgRI,1)
                latest = time.Now()
/*		latency := 40
                        switch  bs {
                        case 1 :
                                latency = 27
                        case 2 :
                                latency = 36
                        case 3 :
                                latency = 44
                        case 4 :
                                latency = 53
                        case 5 :
                                latency = 62
                        case 6 :
                                latency = 70
                        case 7 :
                                latency = 81
                        case 8 :
                                latency = 91
                        }
*/
		tick := time.Tick(time.Duration(t) * time.Millisecond)
                for i=1;i<bs;i++{
			select {
			case	jobs[i] = <-c:
				avgRI = (9 * avgRI + 1 * (int(time.Since(latest))/1000000))/10
				avgRI = max(avgRI,1)
				latest = time.Now()
				j = j+1
		//		log.Println(j)
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
//		go argusBackend(jobs,i)
		mut.Lock()
		totalBatches++
		totalRequests = totalRequests + uint64(i)
		mut.Unlock()
		}
	}
}
func min(a, b int) int {
    if a <= b {
        return a
    }
    return b
}
func max(a, b int) int {
    if a >= b {
        return a
    }
    return b
}
func stage2(j job, count int){
//	log.Println(time.Duration(count)*time.Millisecond)
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
		//	log.Print("gpu request# ",job_num, "bSize: ",currjob.batchsize)
			switch  currjob.batchsize {
			case 1 :
				throughput = 10
				latency = 21
			case 2 :
				throughput = 11
				latency = 25
			case 3 :
				throughput = 12
				latency = 26
			case 4 :
				throughput = 13
				latency = 29
			case 5 :
				throughput = 14
				latency = 33
			case 6 :
				throughput = 36
				latency = 33
			case 7 :
				throughput = 17
				latency = 36
			case 8 :
				throughput = 18
				latency = 38
			case 9 :
				throughput = 20
				latency = 41
			}
		//	log.Println(time.Duration(throughput)*time.Millisecond)
			time.Sleep((time.Duration(throughput) * time.Millisecond))
			go stage2(currjob, latency-throughput)
	}
}

var zmqChan chan []string
func frontToBack() {
        context, _ := zmq.NewContext()
        //defer context.Close()
        zsock, _ := context.NewSocket(zmq.DEALER)
        defer zsock.Close()
	zsock.Connect("tcp://localhost:10090")
	zmqChan = make (chan []string)
	//msg, _ := zsock.RecvMessage(0)
	//log.Println(len(msg))
	//go backToFront(zsock)
	go backToFront(zsock)
	for true {
		log.Println("waiting for new req")
		s:=<-zmqChan
		log.Println("sending new req")
		zsock.SendMessage(s)
		log.Println("sent new req")
	}

}
func backToFront(zsock* zmq.Socket) {
	for true {
		msg, _ := zsock.RecvMessage(0)
		log.Println("waiting to receive from ZMQ")
		for i:=0;i<len(msg);i++{
			var fres [1000]float32
			arr := []byte(msg[i])
			buffferReceived := bytes.NewReader(arr)
			err := binary.Read(buffferReceived , binary.LittleEndian, &fres)
			if err != nil {
				log.Println("binary.Read failed:", err)
			}
			log.Println("sending to waiting channel")
			ch:= <-resChan
			ch<-string(len(fres))
		}
	}
}

var resChan = make (chan chan string, 900)
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
//      log.Println("Received from Argus Backend")
//      log.Println(len(msg))
        for i:=0;i<count;i++ {
        //      zsock.Send(string(jobs[i].buf),Flag(C.ZMQ_SNDMORE))
        //      log.Println(len(msg[i]))
        //      log.Printf("type is %T",msg[i])
        var fres [1000]float32
        arr := [] byte(msg[i])
//      for i:=0 ; i < 4000 ;i++ {
//              log.Print(arr[i])
//      }
        buffferReceived := bytes.NewReader(arr)
        err := binary.Read(buffferReceived , binary.LittleEndian, &fres)
        if err != nil {
                log.Println("binary.Read failed:", err)
        }
                jobs[i].ch <- string(len(fres))
                //log.Println(msg[i])
        }
}

func processbatch(jobs [9]job, count int ) {
	buf1 := jobs[0].buf
	res_chan := make (chan string)
//	log.Print("sending to gpu and count is ", count)
//	log.Println(avgRI)
	gpu_channel <- job{res_chan,buf1,count}
//	log.Print("waiting to recieve from res_chan")
	cstr := <-res_chan
//	log.Print("recieved from res_chan")
	i:=0
        for i=0;i<count;i++ {
		jobs[i].ch <- cstr
	}
}

type incomingReq struct {
	r	*http.Request
	w	http.ResponseWriter
}

var sema = make (chan int, 100)

func receiveThread() {
	for true {
	}
}
func preprocess(w http.ResponseWriter, r *http.Request) {
	im, err := jpeg.Decode(r.Body)
//	log.Print("preprocess")
        if err != nil {
		log.Print("Decode Failed, returning")
               http.Error(w, err.Error(), http.StatusBadRequest)
      		return
        }
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
	//log.Print("done preprocess")
	
	tc--
//	log.Print(tc)
	<-sema
}


func modclass1(w http.ResponseWriter, r *http.Request) {
/*        buffer, err := ioubtil.ReadAll(r.Body)
        if err != nil {
                http.Error(w, err.Error(), http.StatusBadRequest)
        return
        }
	log.Print(len(buffer))*/
	tc++
//	log.Print(tc)
	sema <- tc
	preprocess(w,r)
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
	tc = 0
	go mainloop()//starts the main loop which receives the requests and batches them
//	go frontToBack() 
	go dummygpu()//concurrent dummy gpu running
	log.Println("Adding REST endpoint /api/classify")
	log.Println("Starting server listening on :8002 with SIM backend")
	log.Fatal(srv.ListenAndServe())
 }

