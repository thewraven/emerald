package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
	//	"github.com/fsnotify/fsnotify"
)

var (
	errBadFormat = errors.New("file is an invalid jade->js")
	rootDir      = flag.String("dir", os.Getenv("JADE_RENDER"), "directory with jade->js files")
	gulpTask     = flag.String("gulp", "", "calls 'gulp _task_'")
	gulpEndTask  = flag.String("gulp-end", "", "calls 'gulp _task_' at the end of emerald execution")
)

func workDir(root, gulpTask, gulpEndTask string) {
	fmt.Printf("[%s] Parsing files in %s\n ", time.Now(), *rootDir)
	if gulpTask != "" {
		callGulp(gulpTask)
		fmt.Println("Gulp task completed")
	}
	descriptors, err := ioutil.ReadDir(*rootDir)
	checkErr(err)
	wg := sync.WaitGroup{}
	wg.Add(len(descriptors))
	for _, descr := range descriptors {
		go func(descr os.FileInfo) {
			if descr.IsDir() {
				return
			}
			path := filepath.Join(*rootDir, descr.Name())
			splitNameIdx := strings.LastIndex(descr.Name(), ".")
			if splitNameIdx != -1 {
				actualName := descr.Name()[:splitNameIdx]
				//replace - from file name
				actualName = strings.Replace(actualName, "-", "_", -1)
				err = processFile(path, actualName)
			} else {
				err = processFile(path, descr.Name())
			}
			if err == errBadFormat {
				return
			}
			checkErr(err)
			wg.Done()
		}(descr)
	}
	wg.Wait()
	if gulpEndTask != "" {
		callGulp(gulpEndTask)
		fmt.Println("Gulp end task completed")
	}
	fmt.Println("Everything OK!")
}

func main() {
	flag.Parse()
	if *rootDir == "" {
		fmt.Println("Params: -dir [directory] or $JADE_RENDER must be set")
		os.Exit(-1)
	}
	initial := time.Now()
	workDir(*rootDir, *gulpTask, *gulpEndTask)
	fmt.Println(time.Since(initial))
}

func callGulp(task string) {
	cmdGulp := exec.Command("gulp", task)
	err := cmdGulp.Run()
	if err != nil {
		fmt.Printf("Error on gulp call %s \n", err.Error())
	}
}

func checkErr(err error) {
	if err != nil {
		fmt.Printf("Aborting for error: %s\n", err.Error())
		panic(err)
	}
	return
}

func processFile(path, fileName string) error {
	fileBuf := bytes.Buffer{}
	fileBuf.Reset()
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	io.Copy(&fileBuf, f)
	if err != nil {
		return err
	}
	f.Close()
	content := fileBuf.String()
	content = strings.Replace(content, `("`, "(`", 1)
	content = strings.Replace(content, `")`, "`)", 1)
	content = strings.Replace(content, "template(", "template_"+fileName+"(", 1)
	err = os.Remove(path)
	if err != nil {
		return err
	}
	f, err = os.Create(path)
	if err != nil {
		return err
	}
	fmt.Fprint(f, content)
	err = f.Close()
	if err != nil {
		return err
	}
	return nil
}
