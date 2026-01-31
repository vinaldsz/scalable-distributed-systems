package main

import (
	"bufio"
	"fmt"
	"os"
	"time"
)

func writeUnbuffered(path string, lines int) (time.Duration, error) {
    f, err := os.Create(path)
    if err != nil {
        return 0, err
    }
    defer f.Close()

    start := time.Now()
    for i := 0; i < lines; i++ {
        if _, err := f.Write([]byte(fmt.Sprintf("line %d\n", i))); err != nil {
            return 0, err
        }
    }
    return time.Since(start), nil
}

func writeBuffered(path string, lines int) (time.Duration, error) {
    f, err := os.Create(path)
    if err != nil {
        return 0, err
    }
    defer f.Close()

    w := bufio.NewWriter(f)
    start := time.Now()
    for i := 0; i < lines; i++ {
        if _, err := w.WriteString(fmt.Sprintf("line %d\n", i)); err != nil {
            return 0, err
        }
    }
    if err := w.Flush(); err != nil {
        return 0, err
    }
    return time.Since(start), nil
}

func main() {
    const lines = 100000

    unbuf, err := writeUnbuffered("unbuffered.txt", lines)
    if err != nil {
        fmt.Println("unbuffered error:", err)
        return
    }

    buf, err := writeBuffered("buffered.txt", lines)
    if err != nil {
        fmt.Println("buffered error:", err)
        return
    }

    fmt.Println("Unbuffered:", unbuf)
    fmt.Println("Buffered:  ", buf)
}
