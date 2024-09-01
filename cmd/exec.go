package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"wget/flag"
	"wget/net"

	"github.com/spf13/cobra"
	"github.com/vbauerster/mpb"
)

func init() {
	flag.SetupFlagName()
	rootCmd.Flags().StringVarP(flag.Output, flag.GetFlagName(flag.OUTPUT_FLAG), "O", "", "Save the downloaded file under a different name")
	rootCmd.Flags().StringVarP(flag.Path, flag.GetFlagName(flag.PATH_FLAG), "P", "", "Specify the directory to save the downloaded file")
	rootCmd.Flags().StringVar(flag.RateLimit, flag.GetFlagName(flag.RATELIMIT_FLAG), "", "Limit the download speed (e.g., 400k or 2M)")
	rootCmd.Flags().BoolVarP(flag.Background, flag.GetFlagName(flag.BACKGROUND_FLAG), "B", false, "Download the file in the background")
	rootCmd.Flags().StringVarP(flag.Input, flag.GetFlagName(flag.INPUT_FLAG), "i", "", "Downloading different files should be possible asynchronously")
}

var rootCmd = &cobra.Command{
	Use:   "wget",
	Short: "A wget clone implemented in Go",
	Long:  `This project aims to recreate some functionalities of wget using the Go programming language.`,
	Args: func(cmd *cobra.Command, args []string) error {
		flag.InitFlagValues()
		if len(args) == 0 && *flag.GetFlagValue(flag.INPUT_FLAG).(*string) == "" {
			return fmt.Errorf("invalid argument")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		fn := Exec(cmd, args)
		fn()
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

func Exec(cmd *cobra.Command, args []string) func() {
	flag.SetupUrls(args)
	if flag.Provided(flag.BACKGROUND_FLAG) {
		return func() {
			runInBackground(args)
		}
	}

	return func() {
		var wg sync.WaitGroup
		p := mpb.New(mpb.WithWaitGroup(&wg)) // Create a progress container with the WaitGroup

		wg.Add(len(flag.GetUrls()))
		for _, url := range flag.GetUrls() {
			go func(url string) {
				defer wg.Done()
				defaultExec(p, url)
			}(url)
		}
		wg.Wait()
	}
}

func defaultExec(p *mpb.Progress, url string) {
	_, err := net.GetWithSpeedLimit(p, url, flag.GetRateLimit())
	if err != nil {
		fmt.Println(err)
	}
}

func runInBackground(args []string) {
	cmd := exec.Command(os.Args[0], args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	err := cmd.Start()
	if err != nil {
		panic(err)
	}
	fmt.Println("Running in background with PID", cmd.Process.Pid)
	fmt.Println("Output will be written in wget-log", cmd.Process.Pid)
	os.Exit(0)
}
