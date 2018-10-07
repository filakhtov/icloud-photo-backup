package signal_handler

import (
	"os"
	"os/signal"
	"syscall"
)

type SignalHandler interface {
	Continue()
	ShouldContinue() bool
	Stop()
	Wait()
}

type signalHandler struct {
	channel chan bool
}

func (sh signalHandler) Stop() {
	if sh.ShouldContinue() {
		close(sh.channel)
	}
}

func (sh signalHandler) Continue() {
	if sh.ShouldContinue() {
		sh.channel <- true
	}
}

func (sh signalHandler) ShouldContinue() bool {
	select {
	default:
		return true
	case <-sh.channel:
		return false
	}
}

func (sh signalHandler) Wait() {
	if sh.ShouldContinue() {
		<-sh.channel
	}

}

func New() (sh SignalHandler) {
	sh = signalHandler{channel: make(chan bool)}
	gracefulStop := make(chan os.Signal)

	signal.Notify(gracefulStop, os.Interrupt, syscall.SIGTERM)

	go handleSignal(gracefulStop, sh)

	return
}

func handleSignal(gracefulStop chan os.Signal, sh SignalHandler) {
	defer func() {
		signal.Stop(gracefulStop)
		close(gracefulStop)
	}()

	<-gracefulStop

	sh.Stop()
}
