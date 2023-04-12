package trap // import "github.com/docker/docker/cmd/dockerd/trap"

import (
	"fmt"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
)

// Trap sets up a simplified signal "trap", appropriate for common
// behavior expected from a vanilla unix command-line tool in general
// (and the Docker engine in particular).
//
//   - If SIGINT or SIGTERM are received, `cleanup` is called, then the process is terminated.
//   - If SIGINT or SIGTERM are received 3 times before cleanup is complete, then cleanup is
//     skipped and the process is terminated immediately (allows force quit of stuck daemon)
func Trap(cleanup func(), logger interface {
	Info(args ...interface{})
}) {
	c := make(chan os.Signal, 1)
	// we will handle INT, TERM here
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		interruptCount := uint32(0)
		for sig := range c {
			go func(sig os.Signal) {
				logger.Info(fmt.Sprintf("Processing signal '%v'", sig))
				switch sig {
				case os.Interrupt, syscall.SIGTERM:
					if atomic.LoadUint32(&interruptCount) < 3 {
						// Initiate the cleanup only once
						if atomic.AddUint32(&interruptCount, 1) == 1 {
							// Call the provided cleanup handler
							cleanup()
							os.Exit(0)
						} else {
							return
						}
					} else {
						// 3 SIGTERM/INT signals received; force exit without cleanup
						logger.Info("Forcing docker daemon shutdown without cleanup; 3 interrupts received")
					}
				}
				// for the SIGINT/TERM non-clean shutdown case, exit with 128 + signal #
				os.Exit(128 + int(sig.(syscall.Signal)))
			}(sig)
		}
	}()
}
