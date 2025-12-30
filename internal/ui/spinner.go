package ui

import (
	"bytes"
	"os"
	"os/exec"

	"github.com/charmbracelet/huh/spinner"
)

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
			os.Stderr.Write(stderr.Bytes())
		}
		if stdout.Len() > 0 {
			os.Stderr.Write(stdout.Bytes())
		}
		return err
	}

	return nil
}

// RunCommandWithSpinner はスピナーを表示しながらコマンドを実行する
func RunCommandWithSpinner(message string, name string, args []string, dir string) error {
	return SpinnerWithResult(message, func() error {
		return RunCommandQuiet(name, args, dir)
	})
}

// RunWithSpinner はスピナーを表示しながら処理を実行する
func RunWithSpinner(message string, fn func() error) error {
	return SpinnerWithResult(message, fn)
}

// Spinner はスピナーを実行（エラーなし版）
func Spinner(title string, action func()) error {
	if Quiet {
		action()
		return nil
	}
	return spinner.New().
		Title(title).
		Action(action).
		Run()
}
