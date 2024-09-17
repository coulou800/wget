package cmd

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"wget/flag"
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
	rootCmd.Flags().BoolVarP(flag.Mirror, flag.GetFlagName(flag.MIRROR_FLAG), "", false, "Enables site mirroring to download and locally replicate a complete website, adjusting all internal links for offline navigation. Useful for offline content access and backup.")
	rootCmd.Flags().StringSliceVarP(flag.Reject, flag.GetFlagName(flag.REJECT_FLAG), "R", []string{}, "Define a list of file suffixes to avoid")
	rootCmd.Flags().StringSliceVarP(flag.Excludes, flag.GetFlagName(flag.EXCLUDE_FLAG), "X", []string{}, "Define a list of directory to ignore")

	state.InitNewState()
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
		fmt.Printf("#Start time: %s\n", utils.GetCurrentTime())
		if !flag.IsMirror() {
			fmt.Printf("#Files: %v\n", len(flag.GetUrls()))
		}
		fmt.Printf("\n\n")
		fn()
		fmt.Printf("#End time: %s\n", utils.GetCurrentTime())
		fmt.Printf("\n\n")

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
			// wg.Add(1)
			MirrorExec(p, &wg, flag.GetUrls()[0])
			p.Wait()
			// wg.Wait()
			// println()
		}
	}

	return func() {

		for _, url := range flag.GetUrls() {
			wg.Add(1)
			go func(url string) {
				defer wg.Done()
				defaultExec(p, url)
				// if err != nil {
				// 	fmt.Printf("error: %v\n", err)
				// }
			}(url)
		}
		p.Wait()
	}
}

func defaultExec(p *mpb.Progress, url string) {
	net.GetWithSpeedLimit(p, url, flag.GetRateLimit())
}

func runInBackground() {
	cmd := exec.Command(os.Args[0], flag.GetArgs()...)
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

	go ExtractURLs()
	go processLinks(p, wg)
	go processMirroring(wg)
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
	// if err != nil {
	// 	fmt.Printf("error: %v\n", err)
	// 	return
	// }

	// defer wg.Done() // Ensure to signal completion
}

func processMirroring(wg *sync.WaitGroup) {
	for fileToProcess := range state.GetStates().Mirror.FileToProcess {
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
							n.Attr[i].Val = strings.TrimLeft(relativePath, "/")
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
	doc, err := html.Parse(r)
	if err != nil {
		fmt.Printf("Error parsing HTML: %v\n", err)
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

func ignored(u string) bool {
	parsedUrl, _ := url.Parse(u)
	path := parsedUrl.Path

	// Check if the file extension is in the reject list
	fileExt := strings.TrimPrefix(strings.ToLower(filepath.Ext(parsedUrl.Path)), ".")
	ext := *flag.GetFlagValue(flag.REJECT_FLAG).(*[]string)
	dirToIgnore := *flag.GetFlagValue(flag.EXCLUDE_FLAG).(*[]string)

	if fileExt != "" {
		for _, rejectedExt := range ext {
			// println(rejectedExt)
			if fileExt == rejectedExt {
				return true
			}
		}
	}

	for _, rejectedDir := range dirToIgnore {
		// println(rejectedExt)
		if utils.PathHasDir(rejectedDir, path) {
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

func ExtractURLs() {
	for e := range state.GetStates().Mirror.ReadyToExtract {
		f, _ := os.Open(e.Path)
		// content, _ := io.ReadAll(f)
		links := getLinks(f)
		links = append(links, utils.ExtractURLs(state.GetBaseUrl(), f)...)

		for _, l := range links {
			_, loaded := state.GetVisitedLinks().Load(l)
			if !loaded {
				state.AddLink(l)
			}

		}

		if _, err := html.Parse(f); err != nil {
			continue
		}

		state.AddFileToProcess(e)
	}
}
