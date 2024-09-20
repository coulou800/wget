package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"wget/flag"
	"wget/logger"
	"wget/net"
	"wget/state"
	"wget/utils"

	"sync"

	"github.com/spf13/cobra"
	"github.com/vbauerster/mpb"
	"golang.org/x/net/html"
)

func init() {
	// var ext []string
	flag.SetupFlagName()
	// rootCmd.Flags().StringVarP(flag.RejectedStr, flag.GetFlagName(flag.REJECT_FLAG), "R", "", "Define a list of file suffixes to avoid")
	rootCmd.Flags().StringVarP(flag.Output, flag.GetFlagName(flag.OUTPUT_FLAG), "O", "", "Save the downloaded file under a different name")
	rootCmd.Flags().StringVarP(flag.Path, flag.GetFlagName(flag.PATH_FLAG), "P", "", "Specify the directory to save the downloaded file")
	rootCmd.Flags().StringVar(flag.RateLimit, flag.GetFlagName(flag.RATELIMIT_FLAG), "", "Limit the download speed (e.g., 400k or 2M)")
	rootCmd.Flags().BoolVarP(flag.Background, flag.GetFlagName(flag.BACKGROUND_FLAG), "B", false, "Download the file in the background")
	rootCmd.Flags().StringVarP(flag.Input, flag.GetFlagName(flag.INPUT_FLAG), "i", "", "Downloading different files should be possible asynchronously")
	rootCmd.Flags().BoolVar(flag.Mirror, flag.GetFlagName(flag.MIRROR_FLAG), false, "Enables site mirroring to download and locally replicate a complete website, adjusting all internal links for offline navigation. Useful for offline content access and backup.")
	rootCmd.Flags().StringSliceVarP(flag.Reject, flag.GetFlagName(flag.REJECT_FLAG), "R", []string{}, "Define a list of file suffixes to avoid")
	rootCmd.Flags().StringSliceVarP(flag.Excludes, flag.GetFlagName(flag.EXCLUDE_FLAG), "X", []string{}, "Define a list of directory to ignore")
	rootCmd.Flags().BoolVar(flag.Convert, flag.GetFlagName(flag.CONVERT_FLAG), false, "convert the links so that they can be viewed offline")

	state.InitNewState()
}

var rootCmd = &cobra.Command{
	Use:   "wget",
	Short: "A wget clone implemented in Go",
	Long:  `This project aims to recreate some functionalities of wget using the Go programming language.`,
	Args: func(cmd *cobra.Command, args []string) error {
		flag.InitFlagValues()
		err := flag.CheckFlags()
		if err != nil {
			return err
		}
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

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	os.Exit(0)
}

func Exec(cmd *cobra.Command, args []string) func() {
	var wg sync.WaitGroup
	p := mpb.New(mpb.WithWaitGroup(&wg))
	flag.SetupUrls(args)
	if flag.Provided(flag.BACKGROUND_FLAG) {
		return func() {

			runInBackground()
		}
	}

	if flag.IsMirror() {
		return func() {
			startMsg := fmt.Sprintf("#Start time: %s\n", utils.GetCurrentTime())
			fmt.Println(startMsg)
			// wg.Add(1)
			MirrorExec(p, &wg, flag.GetUrls()[0])
			p.Wait()

			endMsg := fmt.Sprintf("#End time: %s\n", utils.GetCurrentTime())
			fmt.Println(endMsg)
		}
	}

	return func() {
		startMsg := fmt.Sprintf("#Start time: %s\n", utils.GetCurrentTime())
		fmt.Println(startMsg)
		fmt.Printf("#Files: %v\n", len(flag.GetUrls()))

		for _, url := range flag.GetUrls() {
			wg.Add(1)
			go func(url string) {
				defer wg.Done()
				defaultExec(p, url)
			}(url)
		}
		p.Wait()

		endMsg := fmt.Sprintf("#End time: %s\n", utils.GetCurrentTime())
		fmt.Println(endMsg)
	}
}

func defaultExec(p *mpb.Progress, url string) {
	net.GetWithSpeedLimit(p, url, flag.GetRateLimit())
}

func runInBackground() {
	cmd := exec.Command(os.Args[0], flag.GetArgs()...)
	cmd.Stdout = logger.OUT
	cmd.Stderr = nil
	cmd.Stdin = nil
	env := append(os.Environ(), "WGET_BACKGROUND=1")

	cmd.Env = env
	err := cmd.Start()
	if err != nil {
		panic(err)
	}

	pid := cmd.Process.Pid
	fmt.Println("Running in background with PID", pid)
	fmt.Printf("Output will be written in  %s/wget-log\n", *flag.Path)
	os.Exit(0)
}

func MirrorExec(p *mpb.Progress, wg *sync.WaitGroup, u string) {
	parsedUrl, err := url.Parse(u)
	if err != nil {
		os.Stderr.WriteString("invalid url\n")
		os.Exit(1)
	}
	state.SetBaseUrl(parsedUrl)
	host := parsedUrl.Host
	path := filepath.Join(*flag.GetFlagValue(flag.PATH_FLAG).(*string), host)
	err = os.MkdirAll(path, 0755)
	if err != nil {
		os.Stderr.WriteString("cannot create the directory " + err.Error() + "\n")
		os.Exit(1)
	}
	flag.SetOutputPath(path)

	go ExtractURLs(wg)
	go processLinks(p, wg)
	go convertLinks(wg)
	go handleAbort(wg)
	wg.Add(1) // Add to wait group for the recursive function
	go func() {
		// defer wg.Done() // Signal completion when done
		mirrorRecursive(p, u)
	}()
	// Wait for all goroutines to finish
	// wg.Wait()

	// Close channels after all processing is done
	// close(state.GetStates().Mirror.Links)
	// close(state.GetStates().Mirror.FileToProcess)
}

func mirrorRecursive(p *mpb.Progress, u string) {

	if dirIgnored(u) {
		state.Abort(u)
		return
	}

	state.SetVisitedLink(u)
	limiter := state.GetLimiter()
	// Wait for rate limiter before making a request
	err := limiter.Wait(context.Background())
	if err != nil {
		fmt.Printf("Rate limiter error: %v\n", err)
		return
	}

	parsedUrl, err := url.Parse(u)
	if err != nil {
		fmt.Printf("Error parsing URL %s: %v\n", u, err)
		return
	}

	relativePath := parsedUrl.Path
	if relativePath == "" || strings.HasSuffix(relativePath, "/") {
		relativePath = filepath.Join(relativePath, "index.html")
	}

	path := *flag.GetFlagValue(flag.PATH_FLAG).(*string)

	fullPath := filepath.Join(path, relativePath)

	err = os.MkdirAll(filepath.Dir(fullPath), 0755)
	if err != nil {
		fmt.Printf("Error creating directory for %s: %v\n", fullPath, err)
		return
	}

	defaultExec(p, u)
}

func convertLinks(wg *sync.WaitGroup) {
	for fileToProcess := range state.GetStates().Mirror.FileToProcess {
		if !flag.Provided(flag.CONVERT_FLAG) {
			wg.Done()
			continue
		}
		f, err := os.Open(fileToProcess.Path)
		if err != nil {
			// fmt.Printf("error opening file: %v", err)
			wg.Done()
			continue
		}

		fileExt := filepath.Ext(fileToProcess.Path)
		baseUrl := fileToProcess.Url
		doc, err := html.Parse(f)
		if err != nil || fileExt != ".html" {
			// fmt.Printf("error parsing HTML: %v", err)
			wg.Done()
			continue
		}

		var traverse func(*html.Node)
		traverse = func(n *html.Node) {
			if n.Type == html.ElementNode {
				for i, attr := range n.Attr {
					if isLinkAttribute(attr.Key) {
						resolvedUrl, err := baseUrl.Parse(attr.Val)
						if err == nil && utils.IsSameDomain(baseUrl, resolvedUrl.String()) {
							relativePath := resolvedUrl.Path
							if relativePath == "" || strings.HasSuffix(relativePath, "/") {
								relativePath = filepath.Join(relativePath, "index.html")
							}
							fname := strings.TrimLeft(relativePath, "/")
							if filepath.Ext(fname) == "" {
								fname += ".html"
							}
							n.Attr[i].Val = fname
						}
					}
				}
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				traverse(c)
			}
		}
		outputPath := fileToProcess.Path
		traverse(doc)

		err = os.MkdirAll(filepath.Dir(outputPath), 0755)
		if err != nil {
			// fmt.Printf("error creating directories for %s: %v", outputPath, err)
			wg.Done()

			continue
		}

		file, err := os.OpenFile(outputPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			// fmt.Printf("error opening file %s: %v", outputPath, err)
			wg.Done()

			continue
		}

		// Write the modified HTML
		err = html.Render(file, doc)
		if err != nil {
			// fmt.Printf("error writing HTML to file %s: %v", outputPath, err)
			wg.Done()

			continue
		}
		file.Close()

		utils.ReplaceURLsInFile(fileToProcess.Path)

		wg.Done()
	}
}

func getLinks(r io.Reader) []string {
	content, _ := io.ReadAll(r)
	doc, err := html.Parse(bytes.NewReader(content))
	if err != nil {
		// fmt.Printf("Error parsing HTML: %v\n", err)
		return nil
	}

	var links []string
	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode {
			for _, attr := range n.Attr {
				if isLinkAttribute(attr.Key) {
					links = append(links, attr.Val)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}

	traverse(doc)

	links = append(links, utils.ExtractURLs(state.GetBaseUrl(), content)...)
	return links
}

func isLinkAttribute(attr string) bool {
	linkAttributes := []string{"src", "href", "data", "poster"}
	for _, a := range linkAttributes {
		if strings.EqualFold(attr, a) {
			return true
		}
	}
	return false
}

func processLinks(p *mpb.Progress, wg *sync.WaitGroup) {
	baseUrl := state.GetBaseUrl()
	for link := range state.GetStates().Mirror.Links {
		absoluteLink := utils.ResolveLink(baseUrl, link)

		if absoluteLink != "" && utils.IsSameDomain(baseUrl, absoluteLink) {
			wg.Add(1)
			go func(link string) {
				// defer wg.Done() // Ensure to signal completion
				mirrorRecursive(p, link)
			}(absoluteLink)
		}
	}
}

func ExtractURLs(wg *sync.WaitGroup) {
	for e := range state.GetStates().Mirror.ReadyToExtract {
		f, _ := os.Open(e.Path)
		// content, _ := io.ReadAll(f)
		links := getLinks(f)

		for _, l := range links {
			l = utils.ResolveLink(state.GetBaseUrl(), l)
			_, loaded := state.GetVisitedLinks().Load(l)
			if !loaded {
				state.AddLink(l)
			}

		}

		if _, err := html.Parse(f); err != nil {
			wg.Done()
			continue
		}

		state.AddFileToProcess(e)
	}
}

func handleAbort(wg *sync.WaitGroup) {
	for range state.GetStates().Aborted {
		wg.Done()
	}
}

func dirIgnored(s string) bool {
	parsedUrl, _ := url.Parse(s)
	dirToIgnore := *flag.GetFlagValue(flag.EXCLUDE_FLAG).(*[]string)
	for _, rejectedDir := range dirToIgnore {
		// println(rejectedExt)
		if utils.PathHasDir(rejectedDir, parsedUrl.Path) {
			return true
		}
	}
	return false
}
