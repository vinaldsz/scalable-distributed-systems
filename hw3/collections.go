package main

import (
	"fmt"
	"sync"
)

func main() {
    m := make(map[int]int)

    var wg sync.WaitGroup

    for g := range 50 {
        wg.Add(1)
        go func(g int) {
            defer wg.Done()
            for i := range 1000 {
                m[g*1000+i] = i
            }
        }(g)
    }

    wg.Wait()

    fmt.Println("len(m):", len(m))
}
