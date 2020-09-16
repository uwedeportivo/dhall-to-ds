package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/inconshreveable/log15"
	flag "github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)


var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

var (
	recordFile string
	outputDir  string
	printHelp    bool
	printVersion bool
)

func init() {
	flag.StringVarP(&recordFile, "record", "r", "", "(required) record file")
	flag.StringVarP(&outputDir, "output", "o", "", "(required) directory to output")
	flag.BoolVarP(&printHelp, "help", "h", false, "print usage instructions")
	flag.BoolVar(&printVersion, "version", false, "print version information")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of dhall-to-ds: \n")
		fmt.Fprintln(os.Stderr, "OPTIONS:")
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr, usageArgs())
	}
}

func usageArgs() string {
	b := bytes.Buffer{}
	w := tabwriter.NewWriter(&b, 0, 8, 1, ' ', 0)

	w.Flush()

	return fmt.Sprintf("ARGS:\n%s", b.String())
}

func versionString(version, commit, date string) string {
	b := bytes.Buffer{}
	w := tabwriter.NewWriter(&b, 0, 8, 1, ' ', 0)

	fmt.Fprintf(w, "version:\t%s", version)
	fmt.Fprintln(w, "")
	fmt.Fprintf(w, "commit:\t%s", commit)
	fmt.Fprintln(w, "")
	fmt.Fprintf(w, "build date:\t%s", date)
	w.Flush()

	return b.String()
}

func logFatal(message string, ctx ...interface{}) {
	log15.Error(message, ctx...)
	os.Exit(1)
}

func dhallToYaml(file string, wr io.Writer) error {
	cmd := exec.Command("dhall-to-yaml","--file", file)
	cmd.Stderr = os.Stderr
	cmd.Stdout = wr
	return cmd.Run()
}

func readRecord(file string) (map[string]interface{}, error) {
	var buf bytes.Buffer

	err := dhallToYaml(file, &buf)
	if err != nil {
		return nil, err
	}

	decoder := yaml.NewDecoder(&buf)

	var rec map[string]interface{}
	err = decoder.Decode(&rec)
	if err != nil {
		return nil, err
	}

	return rec, nil
}

func writeYaml(filename string, contents interface{}) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	bw := bufio.NewWriter(f)
	defer bw.Flush()

	e := yaml.NewEncoder(bw)

	return  e.Encode(contents)
}

func main() {
	log15.Root().SetHandler(log15.StreamHandler(os.Stdout, log15.LogfmtFormat()))

	flag.Parse()

	if printHelp {
		flag.Usage()
		os.Exit(0)
	}

	if printVersion {
		output := versionString(version, commit, date)
		fmt.Fprintln(os.Stderr, output)
		os.Exit(0)
	}

	if recordFile == "" {
		flag.Usage()
		os.Exit(1)
	}

	if outputDir == "" {
		flag.Usage()
		os.Exit(1)
	}

	err := os.MkdirAll(outputDir, 0777)
	if err != nil {
		logFatal("failed to establish output dir", "output", outputDir, "error", err)
	}

	rec, err := readRecord(recordFile)
	if err != nil {
		logFatal("failed to read record", "record", recordFile, "error", err)
	}

	for comp, compRec := range rec {
		compRecM, ok := compRec.(map[string]interface{})
		if !ok {
			logFatal("unexpected component member", "component", comp)
		}
		err := os.MkdirAll(filepath.Join(outputDir, strings.ToLower(comp)), 0777)
		if err != nil {
			logFatal("failed to create component output directory", "component", comp, "error", err)
		}

		for kind, kindRec := range compRecM {
			kindRecM, ok := kindRec.(map[string]interface{})
			if !ok {
				logFatal("unexpected kind member", "component", comp, "kind", kind)
			}

			for name, contents := range kindRecM {
				filename := filepath.Join(outputDir, strings.ToLower(comp), fmt.Sprintf("%s.%s.yaml", name, kind))

				err := writeYaml(filename, contents)
				if err != nil {
					logFatal("failed to write yaml", "file", filename, "error", err)
				}
			}
		}
	}

	log15.Info("Done")
}
