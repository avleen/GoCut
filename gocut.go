package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"syscall"
)

var options struct {
	Cpuprof       string
	Outfile       string
	Leadingbytes  int
	Trailingbytes int
}

func init() {
	flag.StringVar(&options.Outfile, "outfile", "", "Location to save log lines")
	flag.StringVar(&options.Cpuprof, "cpuprofile", "", "Write CPU profile to disk")
	flag.IntVar(&options.Leadingbytes, "leadingbytes", 0, "Number of leading bytes to trim")
	flag.IntVar(&options.Trailingbytes, "trailingbytes", 0, "Number of trailing bytes to trim")
	flag.Parse()

	// Set the number of threads to use
	runtime.GOMAXPROCS(10)
}

func cut_bytes(line_chan chan []byte, save_chan chan []byte) {
	// At the end of the method, close save_chan
	defer close(save_chan)

	for {
		line, more := <-line_chan
		if more {
			if options.Leadingbytes > 0 {
				line = line[options.Leadingbytes:]
			}
			if options.Trailingbytes > 0 {
				line = line[:options.Trailingbytes]
			}
			save_chan <- line
		} else {
			break
		}
	}
}

func save_file(save_chan chan []byte, done_chan chan bool, outfile string) {
	// At the end of the method, close done_chan
	defer close(done_chan)

	var fh *os.File
	if outfile == "-" {
		fh = os.Stdout
	} else {
		fh_opened, fh_err := os.OpenFile(outfile, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
		if fh_err != nil {
			log.Fatal("Could not open file for writing: ", fh_err)
		}
		fh = fh_opened
	}
	defer fh.Close()

	// Make this a buffered writer, it's more efficient, reduces system CPU
	// significantly
	bfh := bufio.NewWriter(fh)
	for {
		line, more := <-save_chan
		if more {
			bfh.Write(line)
			bfh.WriteString("\n")
		} else {
			break
		}
	}

	// Flush out any remaining buffered writes
	if err := bfh.Flush(); err != nil {
		log.Print("Couldn't flush output to buffered file: ", err)
	}
}

func main() {
	// Some stuff to capture signals and do cleanup on exit
	go func() {
		sigchan := make(chan os.Signal, 10)
		signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)
		<-sigchan
		if options.Cpuprof != "" {
			pprof.StopCPUProfile()
		}
		os.Exit(0)
	}()
	// Enable CPU profiling if needed
	if options.Cpuprof != "" {
		f, err := os.Create(options.Cpuprof)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	// Make the channels we need
	save_chan := make(chan []byte)
	line_chan := make(chan []byte)
	done_chan := make(chan bool)

	go cut_bytes(line_chan, save_chan)
	go save_file(save_chan, done_chan, options.Outfile)

	scanner := bufio.NewScanner(os.Stdin)
	// Some of the lines in the file are long, and cause an error.
	// Let's increase the buffer to 1Mb
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)
	for scanner.Scan() {
		line_chan <- scanner.Bytes()
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "reading standard input:", err)
	}

	// When we close this, the loop in the cut_bytes function will break,
	// and close down save_chan.
	close(line_chan)
	// Wait for done_chan to close
	<-done_chan
}
