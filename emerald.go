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
	"strings"
	"time"
	//	"github.com/fsnotify/fsnotify"
)

var (
	fileBuf      bytes.Buffer
	errBadFormat = errors.New("file is an invalid jade->js")
	rootDir      = flag.String("dir", os.Getenv("JADE_RENDER"), "directory with jade->js files")
	gulpTask     = flag.String("gulp", "", "calls 'gulp _task_'")
	gulpEndTask  = flag.String("gulp-end", "", "calls 'gulp _task_' at the end of emerald execution")
	parent       = flag.String("watch", "", "directory to watch for changes")
)

func workDir(root, gulpTask, gulpEndTask string) {
	fmt.Printf("[%s] Parsing files in %s\n ", time.Now(), *rootDir)
	if gulpTask != "" {
		callGulp(gulpTask)
		fmt.Println("Gulp task completed")
	}
	descriptors, err := ioutil.ReadDir(*rootDir)
	checkErr(err)
	for _, descr := range descriptors {
		if descr.IsDir() {
			continue
		}
		path := *rootDir + "/" + descr.Name()
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
			//fmt.Printf("Omitting %s, is not a valid jade->js file\n", path)
			continue
		}
		checkErr(err)
		fmt.Printf("Parse of %s OK\n", path)
	}
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
	workDir(*rootDir, *gulpTask, *gulpEndTask)
	if *parent == "" {
		return
	}
	/*	watcher, err := fsnotify.NewWatcher()
		if err != nil {
			log.Fatal(err)
			os.Exit(-1)
		}
		defer watcher.Close()
		wg := new(sync.WaitGroup)
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-watcher.Events:
					workDir(*rootDir, *gulpTask, *gulpEndTask)
				case err := <-watcher.Errors:
					log.Println("ERROR", err)
				}
			}
		}()
		err = watcher.Add(*parent)
		wg.Wait()
		if err != nil {
			log.Fatal(err)
		}
		os.Exit(1)*/
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
	idx := strings.Index(content, `("`)
	if idx == -1 {
		return errBadFormat
	}
	idx = strings.Index(content, `")`)
	if idx == -1 {
		return errBadFormat
	}
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
