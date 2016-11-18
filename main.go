package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
	"syscall"
)

const TemplateGoCode = `package main

func main() {
    
}
`
const EditorPath = "/Applications/Visual Studio Code.app"
const TerminalPath = "/Applications/Utilities/Terminal.app"
const ShellPath = "bash"

var logger = log.New(os.Stdout, "gcp", log.Llongfile)

func execute(dir, command string, args ...string) error {
	cmd := exec.Command(command, args...)
	cmd.Dir = dir

	var exitStatus int
	b, err := cmd.CombinedOutput()

	if err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			if s, ok := e.Sys().(syscall.WaitStatus); ok {
				exitStatus = s.ExitStatus()
			} else {
				exitStatus = -1
			}
		}
	} else {
		exitStatus = 0
	}

	if exitStatus != 0 {
		return errors.New("Failed execution (output: " + string(b) + ")")
	}

	return nil
}

func runShell(shell, dir string) error {
	cmd := exec.Command(shell)
	cmd.Dir = dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	var exitStatus int
	err := cmd.Run()

	if err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			if s, ok := e.Sys().(syscall.WaitStatus); ok {
				exitStatus = s.ExitStatus()
			} else {
				exitStatus = -1
			}
		}
	} else {
		exitStatus = 0
	}

	if exitStatus != 0 {
		return errors.New("Failed execution of shell")
	}

	return nil
}

func main() {
	editorPath := os.Getenv("GPC_EDITOR_PATH")

	if len(editorPath) == 0 {
		editorPath = EditorPath
	}

	terminalPath := os.Getenv("GPC_TERMINAL_PATH")

	if len(terminalPath) == 0 {
		terminalPath = TerminalPath
	}

	shellPath := os.Getenv("GPC_SHELL_PATH")

	if len(shellPath) == 0 {
		shellPath = ShellPath
	}

	userID := os.Getenv("GPC_GITHUB_USER_ID")

	if len(userID) == 0 {
		logger.Fatal("You must set \"GPC_GITHUB_USER_ID\" your github id")
	}

	help := flag.Bool("help", false, "Show usage")
	tmp := flag.Bool("tmp", false, "If you want a project for once, use this.(When shell terminate, all files will be removed.)")
	openEditor := flag.Bool("open-editor", true, "Open the project in editor(Using open -a)")
	openNewTerminal := flag.Bool("new-terminal", true, "Open shell in a new window. If tmp is set, this is always false.(Using open -a)")
	flag.Parse()

	if *help {
		flag.Usage()

		return
	}

	if flag.NArg() != 1 {
		logger.Fatalln("Usage is ", flag.Arg(0), "[options]", "[project name]")
	}

	if strings.Index(flag.Arg(0), "/") != -1 {
		logger.Fatalln("Invalid project name")
	}

	root := path.Clean(os.Getenv("GOPATH") + "/src/github.com/" + userID + "/")
	dir := path.Clean(root + "/" + flag.Arg(0))

	if strings.Index(dir, root) != 0 {
		logger.Fatalln("Invalid project name")
	}

	if _, err := os.Stat(dir); err == nil {
		logger.Fatalln("You have already created", flag.Arg(1))
	}

	if err := os.MkdirAll(dir, 0774); err != nil {
		logger.Fatalln("Failed creating directories", err.Error())
	}

	fp, err := os.Create(dir + "/main.go")

	if err != nil {
		logger.Fatalln("Failed creating main.go", err.Error())
	}

	if _, err := fp.Write([]byte(TemplateGoCode)); err != nil {
		fp.Close()
		logger.Fatalln("Failed writing in main.go", err.Error())
	}

	fp.Close()

	if err := execute(dir, "git", "init"); err != nil {
		logger.Fatalln("git initialization", err)
	}

	if *openEditor {
		if err := execute(dir, "open", ".", "-a", EditorPath); err != nil {
			logger.Fatalln("opening editor", err)
		}
	}

	if !*tmp {
		if *openNewTerminal {
			if err := execute(dir, "open", ".", "-a", terminalPath); err != nil {
				logger.Fatalln("opening a new temrinal", err)
			}
		} else {
			if err := runShell(shellPath, dir); err != nil {
				logger.Fatalln(err)
			}
		}
	} else {
		for {
			if err := runShell(shellPath, dir); err != nil {
				logger.Fatalln(err)
			}

			fmt.Printf("All files will be removed. OK?[Y/N]: ")

			var C string
			fmt.Scan(&C)

			if lower := strings.ToLower(C); lower == "y" || lower == "yes" {
				break
			}
		}

		os.RemoveAll(dir)
	}
}
