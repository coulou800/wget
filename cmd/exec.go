package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"wget/flag"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "wget",
	Short: "A wget clone implemented in Go",
	Long:  `This project aims to recreate some functionalities of wget using the Go programming language.`,
	Run: func(cmd *cobra.Command, args []string) {
		fn := Exec(cmd, args)
		fn(args)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	os.Exit(0)
}
func init() {
	rootCmd.Flags().StringVarP(&flag.Output, flag.OUTPUT, "O", "", "Save the downloaded file under a different name")
	rootCmd.Flags().StringVarP(&flag.Path, flag.PATH, "P", "", "Specify the directory to save the downloaded file")
	rootCmd.Flags().StringVar(&flag.RateLimit, flag.RATELIMIT, "", "Limit the download speed (e.g., 400k or 2M)")
	rootCmd.Flags().BoolVarP(&flag.Background, flag.BACKGROUND, "B", false, "Download the file in the background")
	rootCmd.Flags().StringVarP(&flag.Input, flag.INPUT, "i", "", "Downloading different files should be possible asynchronously")
}

func Exec(cmd *cobra.Command, args []string) func(...any) error {
	if flag.Provided(flag.BACKGROUND) {
		return func(...any) error {
			return runInBackground(args)
		}
	}

	return func(...any) error {
		return defaultExec(args)
	}
}

func defaultExec(args []string) error {
	return nil
}

func runInBackground(args []string) error {
	cmd := exec.Command(os.Args[0], args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	err := cmd.Start()
	if err != nil {
		return err
	}
	fmt.Println("Running in background with PID", cmd.Process.Pid)
	fmt.Println("Output will be written in wget-log", cmd.Process.Pid)
	return nil
}
