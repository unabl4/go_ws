package main

import (
  "fmt"
  "sort"
  "sync"  // (wg) waitgroup
  "strconv" // int -> str
  "strings" // join
)

// MD5: 10ms (fast), 1-concurrent
// CRC32: 1s (slow), ∞

func ExecutePipeline(jobs...job) {
  // a channel between two jobs
  var prevChan, currChan chan interface{}

  wg := &sync.WaitGroup{}
  wg.Add(len(jobs)) // +

  for idx, dataJob := range jobs {
    if idx == 0 { // the first job?
      prevChan = nil
      currChan = make(chan interface{})
    } else if idx == len(jobs)-1 {  // last job?
      prevChan = currChan
      currChan = nil  // the end
    } else {
      prevChan = currChan
      currChan = make(chan interface{})
    }

    // spin jobs
    go func(job job, in, out chan interface{}) {
      defer func(ch chan interface{}) {
        wg.Done() // -1
        if ch != nil {
          close(ch)
        }
      }(out)

      job(in, out)
    }(dataJob, prevChan, currChan)
  }

  // exit
  wg.Wait()
}

// ---

func singleHashInner(data string, ch chan interface{}) {
  var arr [3]string // fixed-size array

  wg := &sync.WaitGroup{}
  wg.Add(2) // 2 go-routines

  arr[1] = "~" // the middle element

  // 1st goroutine
  go func() {
    defer wg.Done() // -1
    arr[0] = DataSignerCrc32(data)
  }()

  // 2nd goroutine
  go func() {
    defer wg.Done() // -1
    arr[2] = DataSignerCrc32(DataSignerMd5Proxy(data))
  }()

  wg.Wait() // == 0

  // write the result
  ch <- strings.Join(arr[:], "")
}

// first step
func SingleHash(in, out chan interface{}) {
  wg := &sync.WaitGroup{}

  for val := range in {
    wg.Add(1) // +1
    data := strconv.Itoa(val.(int)) // interface (int) -> int -> string
    go func(data string) {
      defer wg.Done() // -1
      singleHashInner(data, out)
    }(data)
  }

  wg.Wait()
}

func multiHashInner(data string, ch chan interface{}) {
  var th int // iterator index
  var arr [6]string // fixed-size array

  wg := &sync.WaitGroup{}
  wg.Add(6) // 6 goroutines in total

  // combine into a string
  for th=0; th <= 5; th++ {
    go func(i int) {
      defer wg.Done() // -1
      out := DataSignerCrc32(strconv.Itoa(i)+data)
      arr[i] = out
    }(th)
  }

  wg.Wait() // wait for all go-routines to complete (==0)

  // convert the fixed-size array into a slice
  ch <- strings.Join(arr[:], "")
}

// second step
func MultiHash(in, out chan interface{}) {
  wg := &sync.WaitGroup{}

  for rawData := range in {
    wg.Add(1) // +1
    data := rawData.(string) // interface (string) -> string
    go func(data string) {
      defer wg.Done() // -1
      multiHashInner(data, out)
    }(data)
  }

  wg.Wait();
}

// final step
func CombineResults(in, out chan interface{}) {
  var arr []string

  for hash := range in {
    arr = append(arr, hash.(string))
  }

  sort.Strings(arr)
  result := strings.Join(arr, "_")
  out <- result // deliver the final result
}

// ---

// to avoid MD5 'overheat'
var mutex = &sync.Mutex{}
func DataSignerMd5Proxy(data string) string {
  mutex.Lock()
  defer mutex.Unlock()
  return DataSignerMd5(data)
}

// ---

// entry point
func main() {
  arr := [...]int{0,1,1,2,3,5,8}

  ExecutePipeline(
    // input stage (load)
    func(in, out chan interface{}) {
      for _, num := range arr {
        out <- num
      }
    },

    job(SingleHash),
    job(MultiHash),
    job(CombineResults),
    // sink
    func(in, out chan interface{}) {
      for y := range in {
        fmt.Println(y)
      }
    },
  )
}
