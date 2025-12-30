package ui

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/fatih/color"
)

// Spinner はターミナルスピナーを表示する
type Spinner struct {
	message  string
	frames   []string
	interval time.Duration
	stopCh   chan struct{}
	doneCh   chan struct{}
	mu       sync.Mutex
	running  bool
}

// NewSpinner は新しいスピナーを作成する
func NewSpinner(message string) *Spinner {
	return &Spinner{
		message:  message,
		frames:   []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		interval: 80 * time.Millisecond,
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
	}
}

// Start はスピナーを開始する
func (s *Spinner) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	go func() {
		cyan := color.New(color.FgCyan).SprintFunc()
		i := 0
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		for {
			select {
			case <-s.stopCh:
				// カーソルを行頭に移動して行をクリア
				fmt.Printf("\r\033[K")
				close(s.doneCh)
				return
			case <-ticker.C:
				frame := s.frames[i%len(s.frames)]
				fmt.Printf("\r%s %s", cyan(frame), s.message)
				i++
			}
		}
	}()
}

// Stop はスピナーを停止する
func (s *Spinner) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	s.mu.Unlock()

	close(s.stopCh)
	<-s.doneCh
}

// Success はスピナーを停止して成功メッセージを表示する
func (s *Spinner) Success(message string) {
	s.Stop()
	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("%s %s\n", green("✓"), message)
}

// Fail はスピナーを停止して失敗メッセージを表示する
func (s *Spinner) Fail(message string) {
	s.Stop()
	red := color.New(color.FgRed).SprintFunc()
	fmt.Printf("%s %s\n", red("✗"), message)
}

// RunWithSpinner はスピナーを表示しながら処理を実行する
func RunWithSpinner(message string, fn func() error) error {
	spinner := NewSpinner(message)
	spinner.Start()

	err := fn()

	if err != nil {
		spinner.Fail(message)
		return err
	}

	spinner.Success(message)
	return nil
}

// RunCommandQuiet はコマンドを実行し、エラー時のみ出力を表示する
func RunCommandQuiet(name string, args []string, dir string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir

	// 出力をキャプチャ
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	if err != nil {
		// エラー時のみ出力を表示
		if stderr.Len() > 0 {
			fmt.Fprintf(os.Stderr, "\n%s\n", stderr.String())
		}
		if stdout.Len() > 0 {
			fmt.Fprintf(os.Stderr, "%s\n", stdout.String())
		}
		return err
	}

	return nil
}

// RunCommandWithSpinner はスピナーを表示しながらコマンドを実行する
func RunCommandWithSpinner(message string, name string, args []string, dir string) error {
	return RunWithSpinner(message, func() error {
		return RunCommandQuiet(name, args, dir)
	})
}
