/*
Copyright © 2022 Mike Messmore <mike@messmore.org>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.
THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package internal

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

func CD() {
	dir := viper.GetString("root")
	err := os.Chdir(dir)
	if err != nil {
		fmt.Printf("ERROR: Could not change directory to %s\n", dir)
		fmt.Println(err)
		os.Exit(1)
	}
}

func GetBrowser() (string, error) {
	// Fall through config, system defaults, then look at random browsers
	browsers := []string{
		viper.GetString("browser"), //config
		"open",                     //macos default
		"xdg-open",                 //XDG default
		"x-www-browser",            // alternatives default
		"firefox",                  // try firefox?
		"google-chrome-stable",     // try chrome?
		"chromium-browser",         // try chromium?
	}
	browser := ""
	var err error
	for _, cmd := range browsers {
		browser, err = exec.LookPath(cmd)
		if err == nil && browser != "" {
			return browser, nil
		}
	}
	return browser, errors.New("Could not find a way to open a browser")
}

func GetEditor() (string, error) {
	editors := []string{
		viper.GetString("editor"), //config
		os.Getenv("VISUAL"),       // VISUAL env variable
		os.Getenv("EDITOR"),       // EDITOR env variable
		"editor",                  // alternatives default
		"vi",                      // vi's usually there
		"nano",                    // nano's usually there if not
	}
	var err error
	var editor string = ""

	for _, cmd := range editors {
		editor, err = exec.LookPath(cmd)
		if err == nil && editor != "" {
			return editor, nil
		}
	}
	return editor, errors.New("Could not find a text editor")
}

func GetHugo() (string, error) {
	hugo, err := exec.LookPath(viper.GetString("hugo"))
	if err != nil {
		return "", errors.New("Could not find hugo executable")
	}
	return hugo, nil
}

func GetGit() (string, error) {
	git, err := exec.LookPath(viper.GetString("git"))
	if err != nil {
		return "", errors.New("Could not find git executable")
	}
	return git, nil
}

func DisplayHumanConfig() {
	fmt.Println("Paths:")
	fmt.Printf("\tHugo root: %s\n\n", viper.GetString("root"))

	fmt.Println("Programs:")
	hugo, err := GetHugo()
	if err == nil {
		fmt.Printf("\tHugo: %s\n", hugo)
	} else {
		fmt.Printf("\t%v\n", err)
	}
	editor, err := GetEditor()
	if err == nil {
		fmt.Printf("\tEditor: %s\n", editor)
	} else {
		fmt.Printf("\t%v\n", err)
	}
	browser, err := GetBrowser()
	if err == nil {
		fmt.Printf("\tBrowser (launcher): %s\n", browser)
	} else {
		fmt.Printf("\t%v\n", err)
	}
}

func YamlConfig(path string) {
	var err error
	var output *os.File
	type RunningConfig struct {
		Root    string `yaml:"root"`
		Hugo    string `yaml:"hugo"`
		Editor  string `yaml:"editor"`
		Browser string `yaml:"browser"`
	}

	if path == "" {
		output = os.Stdout
	} else {
		output, err = os.Create(path)
		if err != nil {
			fmt.Printf("Error creating %s\n", path)
			fmt.Println(err)
		}
	}

	root := viper.GetString("root")
	hugo, err := GetHugo()
	if err != nil {
		hugo = fmt.Sprint(err)
	}
	editor, err := GetEditor()
	if err != nil {
		editor = fmt.Sprint(err)
	}
	browser, err := GetBrowser()
	if err != nil {
		browser = fmt.Sprint(err)
	}

	rc := RunningConfig{
		Root:    root,
		Hugo:    hugo,
		Editor:  editor,
		Browser: browser,
	}

	y, _ := yaml.Marshal(&rc)

	fmt.Fprint(output, string(y))
}

func PromptToSaveConfig(root string, configPath string) {
	realRoot, _ := filepath.Abs(root)
	viper.Set("root", realRoot)
	DisplayHumanConfig()
	if !Prompt(
		fmt.Sprintf(
			"Save this config to %s?",
			configPath,
		)) {
		fmt.Println("Configuration not saved")
		os.Exit(0)
	}
	YamlConfig(configPath)
	fmt.Printf("Configuration saved to %s\n", configPath)
}

func Prompt(prompt string) bool {
	buf := bufio.NewReader(os.Stdin)

	fmt.Printf("%s [y/n]? ", prompt)
	resp, err := buf.ReadString('\n')
	if err != nil {
		fmt.Printf("Error reading from prompt: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("\n")

	return strings.ToLower(strings.TrimSpace(resp))[0] == 'y'
}

type RunError struct {
	Output   string
	ExitCode int
	Tool     string
}

func (r *RunError) Error() string {
	return fmt.Sprintf("%s exited %d:\n%s",
		r.Tool, r.ExitCode, r.Output,
	)
}

func Run(args ...string) *RunError {
	cmd := args[0]
	cmdArgs := args[1:]
	err := exec.Command(cmd, cmdArgs...).Run()
	if err != nil {
		real_err := err.(*exec.ExitError)
		return &RunError{
			Tool:     args[0],
			Output:   string(real_err.Stderr),
			ExitCode: real_err.ExitCode(),
		}
	}
	return nil
}

func Background(args ...string) (*os.Process, error) {
	procAttr := os.ProcAttr{}
	proc, err := os.StartProcess(args[0], args, &procAttr)
	if err != nil {
		return nil, err
	}
	return proc, nil
}

func Exec(args ...string) error {
	env := os.Environ()
	err := syscall.Exec(args[0], args, env)
	if err != nil {
		return err
	}
	return nil
}
