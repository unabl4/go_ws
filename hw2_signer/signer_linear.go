package main

import (
  "fmt"
  "sort"
  "strconv" // int -> str
  "strings" // join
)

// MD5: 10ms (fast), 1 concurrent
// CRC32: 1s (slow), ∞

// output = combined result
func ExecutePipeline(in... string) string {
  var hashes []string
  for _, data := range in {
    single := SingleHash(data) // first step
    last := MultiHash(single)
    hashes = append(hashes, last)
    // fmt.Println(last)
  }

  // finally
  return CombineResults(hashes)
}

// first step
func SingleHash(data string) string {
  return DataSignerCrc32(data) + "~" + DataSignerCrc32(DataSignerMd5(data))
}

// second step
func MultiHash(data string) string {
  var th int // iterator index
  var out string

  // combine into a string
  for th=0; th <= 5; th++ {
    out += DataSignerCrc32(strconv.Itoa(th)+data)
  }

  return out
}

// final step
func CombineResults(results []string) string {
  sort.Strings(results)
  return strings.Join(results, "_")
}

// ---

// entry point
func main() {
  // fmt.Println(ExecutePipeline("0", "1", "2", "3", "4", "5", "6"))
  fmt.Println(ExecutePipeline("0", "1", "1", "2", "3", "5", "8"))
  // fmt.Println(ExecutePipeline("0", "1"))
  // fmt.Scanln()
}
