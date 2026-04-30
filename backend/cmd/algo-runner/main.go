// Package main is the JSON-line bridge between the Python evaluation
// harness and the production Go baseline algorithm. It reads one
// baseline.Input as JSON per stdin line, runs baseline.Analyze, and
// writes the baseline.Result as a single JSON line to stdout.
//
// The Python harness in evaluation/ launches this binary once per
// algorithm sweep with a long-lived stdin/stdout pipe so per-call startup
// cost is paid once. Errors are logged to stderr and produce an output
// line of {"error": "..."} so the harness can keep going.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"eldercare/backend/internal/baseline"
)

func main() {
	versionFlag := flag.Bool("version", false, "print algorithm version and exit")
	flag.Parse()
	if *versionFlag {
		fmt.Println(baseline.Version)
		return
	}

	in := bufio.NewReader(os.Stdin)
	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()

	dec := json.NewDecoder(in)
	enc := json.NewEncoder(out)

	for {
		var input baseline.Input
		if err := dec.Decode(&input); err != nil {
			if err == io.EOF {
				return
			}
			log.Printf("decode: %v", err)
			_ = enc.Encode(map[string]string{"error": err.Error()})
			_ = out.Flush()
			continue
		}

		result := baseline.Analyze(input)
		if err := enc.Encode(result); err != nil {
			log.Printf("encode: %v", err)
			os.Exit(1)
		}
		if err := out.Flush(); err != nil {
			log.Printf("flush: %v", err)
			os.Exit(1)
		}
	}
}
