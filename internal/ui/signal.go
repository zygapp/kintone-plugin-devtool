package ui

import (
	"os"
	"os/signal"
	"syscall"
)

// SetupSignalHandler はCtrl+Cのグレースフルな終了を設定する
func SetupSignalHandler() chan os.Signal {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	return sigChan
}

// HandleInterrupt はシグナルを受信したら指定の関数を実行する
func HandleInterrupt(cleanup func()) {
	sigChan := SetupSignalHandler()
	go func() {
		<-sigChan
		if cleanup != nil {
			cleanup()
		}
		os.Exit(0)
	}()
}

// WithInterruptHandler はシグナル処理を設定して関数を実行する
func WithInterruptHandler(fn func() error, cleanup func()) error {
	sigChan := SetupSignalHandler()

	done := make(chan error, 1)
	go func() {
		done <- fn()
	}()

	select {
	case <-sigChan:
		if cleanup != nil {
			cleanup()
		}
		return nil
	case err := <-done:
		return err
	}
}
