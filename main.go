// nging-installer [install|upgrade|uninstall] 5.1.0

package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	godl "github.com/admpub/go-download/v2"
	"github.com/admpub/go-download/v2/progressbar"
	"github.com/webx-top/com"
)

var osName = runtime.GOOS
var archName = runtime.GOARCH
var operate = `install`
var version = `5.1.2`
var saveDir string
var softwareURL = `https://img.nging.coscms.com/nging/v%s/`
var binName = "nging"
var fileName string
var fileFullName string
var softwareFullURL string
var workDir string
var local string
var supports = map[string][]string{
	`darwin`:  {`amd64`, `arm64`},
	`linux`:   {`386`, `amd64`, `arm64`, `arm-7`, `arm-6`, `arm-5`},
	`windows`: {`386`, `amd64`},
	`freebsd`: {`amd64`},
}

func parseArgs() {
	flag.StringVar(&local, `local`, ``, `--local ./nging_darwin_amd64.tar.gz`)
	flag.StringVar(&softwareURL, `softwareURL`, softwareURL, `--softwareURL `+softwareURL)
	flag.StringVar(&binName, `name`, binName, `--name `+binName)
	flag.StringVar(&saveDir, `saveDir`, saveDir, `--saveDir `+saveDir)
	defaultUsage := flag.Usage
	flag.Usage = func() {
		defaultUsage()
		fmt.Println()
		fmt.Println(`Command Format:`, os.Args[0], `install|upgrade|up|uninstall|un`, `5.0.0`, `[saveDir]`)
	}
	flag.Parse()

	var extName string
	if osName == `windows` {
		extName = `.exe`
	}
	if len(binName) == 0 {
		binName = strings.TrimSuffix(filepath.Base(os.Args[0]), `-installer`+extName)
	}
	parts := strings.SplitN(binName, `.`, 2)
	if len(parts) == 1 && len(extName) > 0 {
		binName += extName
	}
	name := parts[0]
	if len(saveDir) == 0 {
		saveDir = name
	}
	fileName = fmt.Sprintf("%s_%s_%s", name, osName, archName)
	fileFullName = fileName + ".tar.gz"
	args := make([]string, len(flag.Args()))
	copy(args, flag.Args())
	switch len(args) {
	case 3:
		saveDir = args[2]
		fallthrough
	case 2:
		version = args[1]
		version = strings.TrimPrefix(version, `v`)
		fallthrough
	case 1:
		operate = args[0]
	default:
		flag.Usage()
		os.Exit(0)
	}
}

func verifyOSAndArch() {
	if _, ok := supports[osName]; !ok {
		com.ExitOnFailure(`Unsupported System:`+osName, 1)
	}
	switch archName {
	case `x86_64`, `amd64`:
		archName = "amd64"
	case "i386", "i686":
		archName = "386"
	case "aarch64_be", "aarch64", "armv8b", "armv8l", "armv8", "arm64":
		archName = "arm64"
	case "armv7":
		archName = "arm-7"
	case "armv7l":
		archName = "arm-6"
	case "armv6":
		archName = "arm-6"
	case "armv5":
		archName = "arm-5"
	case "arm":
		armVersion := os.Getenv(`GOARM`)
		switch armVersion {
		case `7`, `6`, `5`:
			archName = "arm-" + armVersion
		default:
			archName = "arm-5"
		}
	}
	if !com.InSlice(archName, supports[osName]) {
		com.ExitOnFailure(`Unsupported Arch:`+archName, 1)
	}
}

func main() {
	verifyOSAndArch()
	parseArgs()
	if len(local) > 0 {
		fmt.Println(`local: `, local)
	}
	if !strings.HasPrefix(softwareURL, `/`) {
		softwareURL += `/`
	}
	softwareFullURL = fmt.Sprintf(softwareURL, version) + fileFullName
	var err error
	workDir, err = filepath.Abs(saveDir)
	if err != nil {
		com.ExitOnFailure(err.Error(), 1)
	}
	switch operate {
	case `un`, `uninstall`:
		uninstall()
	case `up`, `upgrade`:
		upgrade()
	case `install`:
		install()
	default:
		install()
	}
}

func downloadAndExtract() {
	compressedFile := fileFullName
	if len(local) == 0 {
		godlOpt := &godl.Options{}
		progress := progressbar.New(godlOpt, 50)
		defer progress.Wait()

		_, err := godl.Download(softwareFullURL, compressedFile, godlOpt)
		if err != nil {
			com.ExitOnFailure(err.Error(), 1)
		}
	} else {
		compressedFile = local
	}
	err := com.MkdirAll(saveDir, os.ModePerm)
	if err != nil {
		com.ExitOnFailure(err.Error(), 1)
	}
	_, err = com.UnTarGz(compressedFile, saveDir)
	if err != nil {
		com.ExitOnFailure(err.Error(), 1)
	}
	distDir := filepath.Join(saveDir, fileName)
	err = com.CopyDir(distDir, saveDir)
	if err != nil {
		com.ExitOnFailure(err.Error(), 1)
	}
	os.RemoveAll(distDir)
	if len(local) == 0 {
		os.Remove(compressedFile)
	}
	os.Chmod(filepath.Join(saveDir, binName), os.ModePerm)
}

func execServiceCommand(op string, mustSucceed ...bool) error {
	cmd := exec.Command(`./`+binName, `service`, op)
	cmd.Dir = workDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		if len(mustSucceed) == 0 || mustSucceed[0] {
			com.ExitOnFailure(err.Error(), 1)
		}
		return err
	}
	fmt.Println(string(out))
	return err
}

func install() {
	downloadAndExtract()
	execServiceCommand(`install`)
	execServiceCommand(`start`)
	fmt.Println(`🎉 Congratulations! Installed successfully.`)
}

func uninstall() {
	execServiceCommand(`stop`)
	execServiceCommand(`uninstall`)
	fmt.Println(`🎉 Congratulations! Successfully uninstalled.`)
	err := os.RemoveAll(saveDir)
	if err != nil {
		com.ExitOnFailure(err.Error(), 1)
	}
	fmt.Println(`🎉 Congratulations! File deleted successfully.`)
}

func upgrade() {
	execServiceCommand(`stop`)
	execServiceCommand(`stop`, false)
	downloadAndExtract()
	execServiceCommand(`start`)
	fmt.Println(`🎉 Congratulations! Successfully upgraded.`)
}
