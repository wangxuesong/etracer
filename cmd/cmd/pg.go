/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"bytes"
	"encoding/binary"
	"errors"
	"etracer/assets"
	"etracer/internal/module"
	"fmt"
	"github.com/cilium/ebpf/perf"
	"log"
	"math"
	"os"
	"os/signal"
	"syscall"

	"github.com/cilium/ebpf"
	manager "github.com/gojue/ebpfmanager"
	"github.com/spf13/cobra"
	"golang.org/x/sys/unix"
)

var (
	pgPath   string
	funcName string
)

// pgCmd represents the pg command
var pgCmd = &cobra.Command{
	Use:   "pg",
	Short: "跟踪 PostgresSQL",
	Long:  `跟踪 PostgresSQL`,
	Run:   runCommand,
}

func runCommand(cmd *cobra.Command, args []string) {
	byteBuf, err := assets.Asset("internal/bytecode/postgres.o")
	if err != nil {
		log.Printf("error: %v\n", err)
		return
	}

	binaryPath := pgPath
	attachFunc := funcName
	probes := []*manager.Probe{
		{
			Section:          "uprobe/exec_simple_qurey",
			EbpfFuncName:     "postgres_query",
			AttachToFuncName: attachFunc,
			BinaryPath:       binaryPath,
		},
	}

	bpfManager := &manager.Manager{
		Probes: probes,
		Maps: []*manager.Map{
			{
				Name: "events",
			},
		},
	}

	bpfManagerOptions := manager.Options{
		DefaultKProbeMaxActive: 512,

		VerifierOptions: ebpf.CollectionOptions{
			Programs: ebpf.ProgramOptions{
				LogSize: 2097152,
			},
		},

		RLimit: &unix.Rlimit{
			Cur: math.MaxUint64,
			Max: math.MaxUint64,
		},
	}

	if err := bpfManager.InitWithOptions(bytes.NewReader(byteBuf), bpfManagerOptions); err != nil {
		fmt.Printf("couldn't init manager %v.\n", err)
		return
	}

	if err := bpfManager.Start(); err != nil {
		fmt.Printf("couldn't start bootstrap manager %v.\n", err)
		return
	}

	stopper := make(chan os.Signal, 1)
	signal.Notify(stopper, os.Interrupt, syscall.SIGTERM)

	eventsMap, found, err := bpfManager.GetMap("events")
	if err != nil {
		log.Printf("error: %v\n", err)
		return
	}
	if !found {
		log.Printf("cant found map: events")
		return
	}

	rd, err := perf.NewReader(eventsMap, os.Getpagesize())
	if err != nil {
		log.Printf("creating perf event reader: %s", err)
		return
	}
	defer rd.Close()

	go func() {
		// Wait for a signal and close the perf reader,
		// which will interrupt rd.Read() and make the program exit.
		<-stopper
		log.Println("Received signal, exiting program..")

		if err := rd.Close(); err != nil {
			log.Fatalf("closing perf event reader: %s", err)
		}
	}()

	log.Printf("Listening for events..")

	// bpfEvent is generated by bpf2go.
	var event module.PostgresEvent
	for {
		record, err := rd.Read()
		if err != nil {
			if errors.Is(err, perf.ErrClosed) {
				return
			}
			log.Printf("reading from perf event reader: %s", err)
			continue
		}

		if record.LostSamples != 0 {
			log.Printf("perf event ring buffer full, dropped %d samples", record.LostSamples)
			continue
		}

		// Parse the perf event entry into a bpfEvent structure.
		if err := binary.Read(bytes.NewBuffer(record.RawSample), binary.LittleEndian, &event); err != nil {
			log.Printf("parsing perf event: %s", err)
			continue
		}

		log.Printf("pid: %d, func: %s, sql: %s", event.Pid, attachFunc, unix.ByteSliceToString(event.Query[:]))
	}

	fmt.Println("pg called")
}

func init() {
	rootCmd.AddCommand(pgCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// pgCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// pgCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	pgCmd.PersistentFlags().StringVarP(&pgPath, "postgres", "p", "/usr/bin/postgres", "postgres binary file path, use to hook")
	pgCmd.PersistentFlags().StringVarP(&funcName, "funcname", "f", "exec_simple_query", "function name to hook")
}
